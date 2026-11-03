// Command mongoexplain menjalankan explain("executionStats") terhadap koleksi
// student_portfolios dan mem-dump satu dokumen portofolio lengkap — bukti
// tertulis pengganti sesi mongosh/Compass untuk laporan.
//
// Yang dicetak:
//  1. daftar indeks koleksi (nama + definisi)
//  2. explain("executionStats") query {student_id: 6} → harus memakai
//     IXSCAN student_id_unique dengan totalKeysExamined kecil
//  3. dokumen portofolio student_id=6 utuh (bukti isi database)
//
// Pemakaian:
//
//	go run ./tools/mongoexplain
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"students_api/config"
)

func main() {
	_ = godotenv.Load()
	uri := config.GetEnv("MONGO_URI", "mongodb://localhost:27017")
	dbName := config.GetEnv("MONGO_DATABASE", "praktikum_backend")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("gagal connect: %v", err)
	}
	defer func() { _ = client.Disconnect(ctx) }()

	coll := client.Database(dbName).Collection("student_portfolios")
	fmt.Printf("== koneksi: %s (db %s, koleksi student_portfolios) ==\n",
		maskURI(uri), dbName)

	// 1. Daftar indeks.
	fmt.Println("\n[1] INDEXES")
	cur, err := coll.Indexes().List(ctx)
	if err != nil {
		log.Fatalf("gagal list indeks: %v", err)
	}
	var idxDocs []bson.M
	if err := cur.All(ctx, &idxDocs); err != nil {
		log.Fatalf("gagal baca indeks: %v", err)
	}
	for _, d := range idxDocs {
		b, _ := json.Marshal(d)
		fmt.Printf("    %s\n", b)
	}

	// 2. explain executionStats untuk query student_id.
	fmt.Println("\n[2] EXPLAIN executionStats — find({student_id: 6})")
	cmd := bson.D{
		{Key: "explain", Value: bson.D{
			{Key: "find", Value: coll.Name()},
			{Key: "filter", Value: bson.D{{Key: "student_id", Value: 6}}},
		}},
		{Key: "verbosity", Value: "executionStats"},
	}
	var res bson.M
	if err := coll.Database().RunCommand(ctx, cmd).Decode(&res); err != nil {
		log.Fatalf("gagal explain: %v", err)
	}
	printExecStats(res)

	// 3. Dump satu dokumen lengkap.
	fmt.Println("\n[3] DOKUMEN student_id=6 (utuh)")
	var doc bson.M
	if err := coll.FindOne(ctx, bson.M{"student_id": 6}).Decode(&doc); err != nil {
		log.Fatalf("gagal FindOne: %v", err)
	}
	b, _ := json.MarshalIndent(doc, "", "  ")
	fmt.Println(string(b))

	// 4. Jumlah total proyek — sanity check 20.
	total := 0
	for _, p := range doc["projects"].(bson.A) {
		if pm, ok := p.(bson.M); ok {
			if t, ok := pm["title"].(string); ok && len(t) > 0 {
				total++
			}
		}
	}
	fmt.Printf("\n[4] total proyek pada dokumen: %d\n", total)
}

// printExecStats menyarikan bagian penting dari hasil explain agar mudah
// dibaca di laporan.
func printExecStats(res bson.M) {
	qp, _ := res["queryPlanner"].(bson.M)
	if qp == nil {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(b))
		return
	}
	winner, _ := qp["winningPlan"].(bson.M)
	fmt.Printf("    namespace        : %v\n", qp["namespace"])
	fmt.Printf("    winningPlan      : %s\n", describePlan(winner, ""))
	exec, _ := res["executionStats"].(bson.M)
	if exec != nil {
		fmt.Printf("    executionSuccess : %v\n", exec["executionSuccess"])
		fmt.Printf("    nReturned        : %v\n", exec["nReturned"])
		fmt.Printf("    totalKeysExamined: %v\n", exec["totalKeysExamined"])
		fmt.Printf("    totalDocsExamined: %v\n", exec["totalDocsExamined"])
		fmt.Printf("    executionTimeMillis: %v\n", exec["executionTimeMillis"])
	}
}

// describePlan merender pohon rencana eksekusi secara rekursif.
func describePlan(node bson.M, indent string) string {
	if node == nil {
		return "(nil)"
	}
	stage, _ := node["stage"].(string)
	out := stage
	switch stage {
	case "IXSCAN":
		out += fmt.Sprintf(" (%s over %v)",
			node["indexName"], keysOf(node["keyPattern"]))
	case "COLLSCAN":
		if f, ok := node["filter"]; ok {
			out += fmt.Sprintf(" filter=%v", f)
		}
	}
	if child, ok := node["inputStage"].(bson.M); ok {
		out += "\n" + indent + "  └─ " + describePlan(child, indent+"  ")
	}
	if sh, ok := node["shards"].(bson.A); ok && len(sh) > 0 {
		if sm, ok := sh[0].(bson.M); ok {
			out += "\n" + indent + "  └─ " + describePlan(sm, indent+"  ")
		}
	}
	return out
}

func keysOf(v any) string {
	switch t := v.(type) {
	case bson.D:
		out := ""
		for _, e := range t {
			if out != "" {
				out += ", "
			}
			out += fmt.Sprintf("%s:%v", e.Key, e.Value)
		}
		return "{" + out + "}"
	case bson.M:
		b, _ := json.Marshal(t)
		return string(b)
	}
	return fmt.Sprintf("%v", v)
}

// maskURI menyembunyikan kredensial URI agar aman dicetak di laporan.
func maskURI(uri string) string {
	if i := indexByte(uri, '@'); i >= 0 {
		return "mongodb+srv://***:***@" + uri[i+1:]
	}
	return uri
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
