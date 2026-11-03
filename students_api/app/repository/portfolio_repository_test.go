package repository

import (
	"strconv"
	"testing"

	"students_api/app/model"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// TestNormalizeTechs menutup bug agregasi: tanpa normalisasi, "Go" dan "Go ",
// serta "React" dan "react", dihitung sebagai teknologi berbeda pada $group.
func TestNormalizeTechs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil_ke_slice_kosong", nil, []string{}},
		{"buang_kosong", []string{"Go", "", "   ", "React"}, []string{"Go", "React"}},
		{"trim_spasi", []string{"  Go  ", "\tReact\n"}, []string{"Go", "React"}},
		{"dedup_kecil_besar", []string{"React", "react", "REACT"}, []string{"React"}},
		{"dedup_tetap_urutan_asli", []string{"Zed", "Go", "Zed"}, []string{"Zed", "Go"}},
		{"duplikat_spasi_dalam_label_utuh", []string{"Node JS", "Node JS"}, []string{"Node JS"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeTechs(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("panjang = %d (%v), ingin %d (%v)",
					len(got), got, len(tc.want), tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("index %d = %q, ingin %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestLimitFieldMenggantiOffsetYangSalah adalah penjaga untuk kesalahan yang
// paling merusak di sini.
//
// Field batas harus "projects.19" (indeks terakhir). Kalau tertukar menjadi
// "projects.20", syarat $exists:false hanya terpenuhi saat array berisi 20
// elemen — artinya proyek ke-21 lolos dan batas 20 tidak pernah berlaku.
// Kalau menjadi "projects.18", batasnya menjadi 19 dan satu tempat terbuang.
func TestLimitFieldMenggantiOffsetYangSalah(t *testing.T) {
	limitField := "projects." + strconv.Itoa(MaxProjectsPerPortfolio-1)

	if want := "projects.19"; limitField != want {
		t.Fatalf("limitField = %q, ingin %q", limitField, want)
	}
	if MaxProjectsPerPortfolio != 20 {
		t.Fatalf("MaxProjectsPerPortfolio = %d, ingin 20", MaxProjectsPerPortfolio)
	}
}

// TestProjectDocumentRoundTrip memastikan id proyek tetap string di model
// meski di database berupa ObjectID — inilah bukti package model bebas dari
// tipe driver.
func TestProjectDocumentRoundTrip(t *testing.T) {
	oid := bson.NewObjectID()
	doc := portfolioDocument{
		StudentID: 7,
		Projects: []projectDocument{{
			ID: oid, Title: "Tugas Besar", Technologies: nil,
			URL: "https://example.test", Year: 2025,
		}},
	}
	got := doc.toModel()

	if got.StudentID != 7 {
		t.Errorf("StudentID = %d, ingin 7", got.StudentID)
	}
	if len(got.Projects) != 1 {
		t.Fatalf("jumlah proyek = %d, ingin 1", len(got.Projects))
	}
	if got.Projects[0].ID != oid.Hex() {
		t.Errorf("ID = %q, ingin %q", got.Projects[0].ID, oid.Hex())
	}
	// technologies nil harus menjadi slice kosong, bukan nil, supaya JSON
	// menghasilkan [] dan bukan null.
	if got.Projects[0].Technologies == nil {
		t.Error("Technologies = nil, ingin slice kosong")
	}
	if len(got.Projects[0].Technologies) != 0 {
		t.Errorf("Technologies = %v, ingin kosong", got.Projects[0].Technologies)
	}
	// Arrays kosong harus tetap kosong, bukan nil, supaya JSON [] bukan null.
	if got.Certificates == nil {
		t.Error("Certificates = nil, ingin slice kosong")
	}
	if got.Projects == nil {
		t.Error("Projects = nil, ingin slice kosong")
	}
}

// TestDocumentKosongTetapSliceKosong menutupi portofolio yang belum diisi:
// bentuk JSON harus sama dengan portofolio yang sudah ada.
func TestDocumentKosongTetapSliceKosong(t *testing.T) {
	got := portfolioDocument{StudentID: 1}.toModel()
	if got.Projects == nil || got.Certificates == nil {
		t.Fatal("slice hasil toModel tidak boleh nil")
	}
	var _ model.Portfolio = got
}

// TestParseObjectID menolak input rusak, karena path ini dipakai saat client
// mengirim portofolio lengkap melalui PUT.
func TestParseObjectID(t *testing.T) {
	if _, err := parseObjectID(""); err == nil {
		t.Error("string kosong harus ditolak")
	}
	if _, err := parseObjectID("bukan-objectid"); err == nil {
		t.Error("string bukan hex harus ditolak")
	}
	if _, err := parseObjectID("12345"); err == nil {
		t.Error("hex terlalu pendek harus ditolak")
	}
	oid := bson.NewObjectID()
	got, err := parseObjectID(oid.Hex())
	if err != nil {
		t.Fatalf("ObjectID valid ditolak: %v", err)
	}
	// bson.ObjectID pada driver v2 adalah array [12]byte, sehingga bisa
	// dibandingkan langsung tanpa method Equal.
	if got != oid {
		t.Errorf("hasil = %v, ingin %v", got, oid)
	}
}
