package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"students_api/app/model"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	// ErrPortfolioNotFound dipakai ketika portofolio belum pernah dibuat.
	ErrPortfolioNotFound = errors.New("portofolio tidak ditemukan")
	// ErrProjectNotFound dipakai ketika id proyek tidak ada di array projects.
	ErrProjectNotFound = errors.New("proyek tidak ditemukan")
	// ErrProjectLimit dikembalikan ketika array projects sudah berisi 20 elemen.
	ErrProjectLimit = errors.New("jumlah proyek sudah mencapai batas")
	// ErrDuplicatePortfolio muncul bila index unik student_id somehow dilanggar.
	ErrDuplicatePortfolio = errors.New("portofolio sudah ada")
)

// MaxProjectsPerPortfolio membatasi pertumbuhan array projects.
//
// Satu dokumen MongoDB dibatasi 16 MB. Array yang dibiarkan tumbuh tanpa batas
// akan suatu saat menabrak batas itu. Batas 20 dijaga di dua lapis: validasi di
// kode (tag validate) DAN secara atomik lewat syarat "projects.19 tidak ada"
// pada AddProject. Lapisan kedua yang menentukan — ia menutup celah balapan
// yang tidak bisa ditutup oleh lapis pertama.
const MaxProjectsPerPortfolio = 20

// DefaultTechStatLimit adalah jumlah teknologi yang dikembalikan agregasi bila
// client tidak menyebutkannya.
const DefaultTechStatLimit = 10

type PortfolioRepository interface {
	EnsureIndexes(ctx context.Context) error
	Upsert(ctx context.Context, studentID int, p model.Portfolio) (model.Portfolio, error)
	FindByStudent(ctx context.Context, studentID int) (model.Portfolio, error)
	AddProject(ctx context.Context, studentID int, p model.Project) (model.Portfolio, error)
	RemoveProject(ctx context.Context, studentID int, projectID string) (model.Portfolio, error)
	TopTechnologies(ctx context.Context, limit int) ([]model.TechnologyStat, error)
}

// portfolioDocument adalah bentuk portofolio SEBAGAIMANA TERSIMPAN di MongoDB.
//
// Tag bson menentukan nama field di database; tag json tidak ada karena struct
// ini tidak pernah dikirim ke client. Pemisahan ini menjaga package model tetap
// bebas dari tipe driver.
type portfolioDocument struct {
	ID           bson.ObjectID         `bson:"_id,omitempty"`
	StudentID    int                   `bson:"student_id"`
	Headline     string                `bson:"headline"`
	Bio          string                `bson:"bio"`
	Projects     []projectDocument     `bson:"projects"`
	Certificates []certificateDocument `bson:"certificates"`
	CreatedAt    time.Time             `bson:"created_at"`
	UpdatedAt    time.Time             `bson:"updated_at"`
}

type projectDocument struct {
	ID           bson.ObjectID `bson:"id"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"`
	URL          string        `bson:"url"`
	Year         int           `bson:"year"`
	CreatedAt    time.Time     `bson:"created_at"`
}

type certificateDocument struct {
	ID        bson.ObjectID `bson:"id"`
	Name      string        `bson:"name"`
	Issuer    string        `bson:"issuer"`
	IssuedAt  time.Time     `bson:"issued_at"`
	CreatedAt time.Time     `bson:"created_at"`
}

func (d portfolioDocument) toModel() model.Portfolio {
	projects := make([]model.Project, 0, len(d.Projects))
	for _, p := range d.Projects {
		techs := p.Technologies
		if techs == nil {
			techs = []string{}
		}
		projects = append(projects, model.Project{
			ID: p.ID.Hex(), Title: p.Title, Technologies: techs,
			URL: p.URL, Year: p.Year, CreatedAt: p.CreatedAt,
		})
	}
	certs := make([]model.Certificate, 0, len(d.Certificates))
	for _, c := range d.Certificates {
		certs = append(certs, model.Certificate{
			ID: c.ID.Hex(), Name: c.Name, Issuer: c.Issuer,
			IssuedAt: c.IssuedAt, CreatedAt: c.CreatedAt,
		})
	}
	return model.Portfolio{
		StudentID: d.StudentID, Headline: d.Headline, Bio: d.Bio,
		Projects: projects, Certificates: certs,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

type portfolioMongoRepository struct{ coll *mongo.Collection }

func NewPortfolioRepository(db *mongo.Database) PortfolioRepository {
	return &portfolioMongoRepository{coll: db.Collection("student_portfolios")}
}

// EnsureIndexes dijalankan sekali saat aplikasi menyala. CreateMany pada index
// yang sudah ada tidak melakukan apa pun, jadi aman dipanggil berulang kali.
//
// Diferensinya dengan repository lain di proyek ini: di sini index BUKAN
// sekadar armour tambahan. Aturan "satu mahasiswa satu portofolio" dijaga
// index unik student_id, bukan oleh pemeriksaan di kode. Kovensinya jelas —
// aplikasi tidak pernah bisa membuat portofolio kedua untuk student_id yang
// sama, bahkan bila ada dua request bersamaan atau ada kode lain di luar
// proyek ini yang menulis ke collection yang sama.
func (r *portfolioMongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			// ATURAN INTI: satu mahasiswa, tepat satu portofolio.
			// Dijaga database, bukan kode.
			Keys:    bson.D{{Key: "student_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("student_id_unique"),
		},
		{
			// Mendukung pencarian proyek satu mahasiswa lewat $elemMatch,
			// dan_lookup berdasarkan id proyek di dalam array.
			Keys:    bson.D{{Key: "projects.id", Value: 1}},
			Options: options.Index().SetName("projects_id_idx"),
		},
		{
			// Mendukung agregasi statistik teknologi. Tanpa index ini,
			// $unwind pada projects.technologies memindai SELURUH
			// collection dan harus menyortir seluruh hasil di memori —
			// batas 100 MB untuk pipeline yang tidak memakai index.
			Keys:    bson.D{{Key: "projects.technologies", Value: 1}},
			Options: options.Index().SetName("projects_technologies_idx"),
		},
		{
			// Membantu menyaring dokumen yang baru saja diubah.
			Keys:    bson.D{{Key: "updated_at", Value: -1}},
			Options: options.Index().SetName("updated_at_idx"),
		},
	})
	if err != nil {
		return fmt.Errorf("membuat index portfolio: %w", err)
	}
	return nil
}

// Upsert membuat portofolio bila belum ada, atau menggantinya sepenuhnya bila
// sudah ada — inilah makna PUT.
//
// Memakai $set untuk headline/bio/projects/certificates dan $setOnInsert untuk
// student_id. created_at sengaja TIDAK di-$set: waktu pembuatan tidak boleh
// berubah setiap kali portofolio diganti.
func (r *portfolioMongoRepository) Upsert(
	ctx context.Context, studentID int, p model.Portfolio,
) (model.Portfolio, error) {
	projects := make([]projectDocument, 0, len(p.Projects))
	for _, pr := range p.Projects {
		oid, err := parseObjectID(pr.ID)
		if err != nil {
			oid = bson.NewObjectID()
		}
		projects = append(projects, projectDocument{
			ID: oid, Title: pr.Title, Technologies: normalizeTechs(pr.Technologies),
			URL: pr.URL, Year: pr.Year, CreatedAt: orNow(pr.CreatedAt),
		})
	}
	certs := make([]certificateDocument, 0, len(p.Certificates))
	for _, ct := range p.Certificates {
		oid, err := parseObjectID(ct.ID)
		if err != nil {
			oid = bson.NewObjectID()
		}
		issued := ct.IssuedAt
		if issued.IsZero() {
			issued = time.Now().UTC()
		}
		certs = append(certs, certificateDocument{
			ID: oid, Name: ct.Name, Issuer: ct.Issuer,
			IssuedAt: issued.Truncate(time.Millisecond), CreatedAt: orNow(ct.CreatedAt),
		})
	}

	ts := now()
	filter := bson.M{"student_id": studentID}
	update := bson.M{
		"$set": bson.M{
			"headline":     p.Headline,
			"bio":          p.Bio,
			"projects":     projects,
			"certificates": certs,
			"updated_at":   ts,
		},
		"$setOnInsert": bson.M{"student_id": studentID, "created_at": ts},
	}
	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After).
		SetUpsert(true)

	var updated portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if err == nil {
		return updated.toModel(), nil
	}
	if mongo.IsDuplicateKeyError(err) {
		// Index unik student_id menahan percobaan membuat portofolio kedua.
		return model.Portfolio{}, ErrDuplicatePortfolio
	}
	return model.Portfolio{}, fmt.Errorf("menyimpan portofolio: %w", err)
}

func (r *portfolioMongoRepository) FindByStudent(
	ctx context.Context, studentID int,
) (model.Portfolio, error) {
	var doc portfolioDocument
	err := r.coll.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Portfolio{}, ErrPortfolioNotFound
		}
		return model.Portfolio{}, fmt.Errorf("mengambil portofolio: %w", err)
	}
	return doc.toModel(), nil
}

// AddProject menambahkan satu proyek memakai $push dengan syarat atomik
// "projects.19 tidak ada".
//
// Pola ini identik dengan replies.49 pada repository ulasan, hanya batasnya
// berbeda. Mengapa penting: pemeriksaan batas dan penambahan terjadi dalam
// SATU operasi, sehingga dua request yang datang bersamaan tidak dapat
// sama-sama lolos ketika tinggal satu proyek tersisa. Kalau batasnya
// diperiksa lebih dulu lalu di-$push setelahnya, keduanya akan lolos dan
// array berisi 21 elemen.
func (r *portfolioMongoRepository) AddProject(
	ctx context.Context, studentID int, p model.Project,
) (model.Portfolio, error) {
	doc := projectDocument{
		ID: bson.NewObjectID(), Title: p.Title,
		Technologies: normalizeTechs(p.Technologies),
		URL:          p.URL, Year: p.Year, CreatedAt: now(),
	}
	limitField := "projects." + strconv.Itoa(MaxProjectsPerPortfolio-1)

	var updated portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx,
		bson.M{
			"student_id": studentID,
			limitField:   bson.M{"$exists": false},
		},
		bson.M{
			"$push": bson.M{"projects": doc},
			"$set":  bson.M{"updated_at": now()},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if err == nil {
		return updated.toModel(), nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.Portfolio{}, fmt.Errorf("menambah proyek: %w", err)
	}

	// Tidak ada dokumen yang ter-update. Bedakan dua sebab: portofolio memang
	// belum ada, atau projects sudah mencapai batas. Keduanya menghasilkan
	// ErrNoDocuments yang sama, jadi satu CountDocuments dipakai untuk
	// membedakan — menebak akan menghasilkan pesan 404 atau 409 yang salah.
	count, cerr := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if cerr != nil {
		return model.Portfolio{}, fmt.Errorf("memeriksa portofolio: %w", cerr)
	}
	if count == 0 {
		return model.Portfolio{}, ErrPortfolioNotFound
	}
	return model.Portfolio{}, ErrProjectLimit
}

// RemoveProject menghapus satu proyek dengan $pull.
//
// $pull menghapus SEMUA elemen yang cocok. Karena id proyek dibangkitkan
// sendiri oleh server (bson.NewObjectID) dan tidak pernah dikirim client,
// tidak mungkin ada dua proyek dengan id sama dalam satu portofolio — jadi
// $pull di sini selalu menghapus tepat satu elemen.
//
// Syarat "projects.id = oid" pada FILTER itu wajib, bukan tambahan. Tanpa
// syarat ini, FindOneAndUpdate tetap cocok pada dokumen portofolio itu meskipun
// tidak ada proyek dengan id tersebut: $pull tidak menghapus apa pun, tetapi
// operasi tetap melaporkan sukses sehingga pemanggil menerima 200 untuk
// penghapusan yang sebenarnya tidak terjadi.
func (r *portfolioMongoRepository) RemoveProject(
	ctx context.Context, studentID int, projectID string,
) (model.Portfolio, error) {
	oid, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return model.Portfolio{}, ErrProjectNotFound
	}
	var updated portfolioDocument
	err = r.coll.FindOneAndUpdate(ctx,
		bson.M{
			"student_id":  studentID,
			"projects.id": oid,
		},
		bson.M{
			"$pull": bson.M{"projects": bson.M{"id": oid}},
			"$set":  bson.M{"updated_at": now()},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if err == nil {
		return updated.toModel(), nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		// Tidak ada dokumen dengan student_id + projects.id tersebut. Bedakan
		// dua sebab: portofolionya tidak ada, atau proyeknya tidak ada.
		count, cerr := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
		if cerr != nil {
			return model.Portfolio{}, fmt.Errorf("memeriksa portofolio: %w", cerr)
		}
		if count == 0 {
			return model.Portfolio{}, ErrPortfolioNotFound
		}
		return model.Portfolio{}, ErrProjectNotFound
	}
	return model.Portfolio{}, fmt.Errorf("menghapus proyek: %w", err)
}

// TopTechnologies menghitung teknologi paling banyak dipakai di seluruh
// portofolio memakai aggregation pipeline dengan DUA $unwind.
//
// Pipeline:
//
//	[{$unwind: "$projects"}]        // satu dokumen -> satu dokumen per proyek
//	[{$unwind: "$projects.technologies"}]  // -> satu baris per pasangan proyek-teknologi
//	[{$group: {_id: "$projects.technologies", total: {$sum: 1}}}]
//	[{$sort: {total: -1, _id: 1}}]
//	[{$limit: N}]
//
// Dua $unwind wajib karena projects DAN technologies keduanya array. Satu
// $unwind saja akan menyisakan technologies sebagai array utuh — $group akan
// menghitung satu dokumen per proyek, bukan satu per teknologi.
func (r *portfolioMongoRepository) TopTechnologies(
	ctx context.Context, limit int,
) ([]model.TechnologyStat, error) {
	if limit < 1 || limit > 100 {
		limit = DefaultTechStatLimit
	}
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$projects"}},
		{{Key: "$unwind", Value: "$projects.technologies"}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$projects.technologies",
			"total": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "total", Value: -1},
			{Key: "_id", Value: 1}, // tie-break: nama asc, agar hasil stabil
		}}},
		{{Key: "$limit", Value: int64(limit)}},
		{{Key: "$project", Value: bson.M{
			"_id": 0, "technology": "$_id", "total": 1,
		}}},
	}
	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("mengagregasi teknologi: %w", err)
	}
	defer cursor.Close(ctx)

	out := []model.TechnologyStat{}
	for cursor.Next(ctx) {
		var row model.TechnologyStat
		if err := cursor.Decode(&row); err != nil {
			return nil, fmt.Errorf("membaca hasil agregasi: %w", err)
		}
		out = append(out, row)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("membaca agregasi: %w", err)
	}
	return out, nil
}

// ---------- helper internal ----------

func orNow(t time.Time) time.Time {
	if t.IsZero() {
		return now()
	}
	return t.Truncate(time.Millisecond)
}

// normalizeTechs membersihkan daftar teknologi: trim spasi, buang yang kosong,
// dan deduplikasi sambil mempertahankan urutan. Tanpa ini "Go" dan "Go " akan
// dihitung sebagai dua teknologi berbeda pada agregasi.
func normalizeTechs(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, t := range in {
		clean := strings.TrimSpace(t)
		if clean == "" {
			continue
		}
		key := strings.ToLower(clean)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func parseObjectID(id string) (bson.ObjectID, error) {
	if id == "" {
		return bson.ObjectID{}, ErrInvalidID
	}
	return bson.ObjectIDFromHex(id)
}
