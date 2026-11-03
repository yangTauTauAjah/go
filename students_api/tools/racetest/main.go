// Command racetest membuktikan bahwa batas 20 proyek bertahan saat diserang
// secara serentak.
//
// Skenario (persis seperti yang diminta laporan):
//  1. PUT portofolio kosong (reset)
//  2. isi 18 proyek secara berurutan (18 < 20, semua harus 200)
//  3. kirim 10 permintaan POST serentak; sisa tempat hanya 2
//     (indeks 18 dan 19), sehingga hasil yang benar adalah TEPAT 2 sukses
//     dan 8 ditolak 409 PROJECT_LIMIT
//  4. cetak jumlah akhir proyek di database — harus tepat 20
//
// Pemakaian:
//
//	go run ./tools/racetest -url http://localhost:3000 -token <JWT>
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type apiResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Data    struct {
		Projects []struct {
			ID string `json:"id"`
		} `json:"projects"`
	} `json:"data"`
}

func main() {
	baseURL := flag.String("url", "http://localhost:3000", "base URL API")
	token := flag.String("token", "", "JWT access token")
	studentID := flag.Int("student", 6, "id mahasiswa pemilik portofolio")
	flag.Parse()
	if *token == "" {
		fmt.Println("flag -token wajib diisi")
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	auth := "Bearer " + *token
	portfolioURL := fmt.Sprintf("%s/api/v1/students/%d/portfolio", *baseURL, *studentID)
	projectsURL := portfolioURL + "/projects"

	call := func(method, url string, body any) (int, apiResponse) {
		var rdr io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rdr = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, url, rdr)
		req.Header.Set("Authorization", auth)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := client.Do(req)
		if err != nil {
			return 0, apiResponse{}
		}
		defer res.Body.Close()
		var out apiResponse
		_ = json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}

	// Langkah 1: reset portofolio supaya hitungan mulai dari nol.
	status, _ := call("PUT", portfolioURL, map[string]any{
		"headline":     "Backend Developer Pemula",
		"bio":          "Portofolio untuk uji batas 20 proyek secara serentak.",
		"projects":     []any{},
		"certificates": []any{},
	})
	fmt.Printf("[1] PUT reset portofolio            : HTTP %d\n", status)

	// Langkah 2: isi 18 proyek secara berurutan.
	for i := 1; i <= 18; i++ {
		status, body := call("POST", projectsURL, map[string]any{
			"title":        fmt.Sprintf("Proyek Urut %02d", i),
			"technologies": []string{"Go"},
			"year":         2026,
		})
		if status != 200 {
			fmt.Printf("[2] proyek urut %02d GAGAL: HTTP %d %s\n", i, status, body.Code)
			return
		}
	}
	status, body := call("GET", portfolioURL, nil)
	fmt.Printf("[2] terisi %d proyek secara berurutan: HTTP %d\n", len(body.Data.Projects), status)

	// Langkah 3: 10 permintaan serentak memperebutkan 2 tempat tersisa.
	var okCount, conflictCount, otherCount int32
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-start // semua goroutine dilepas bersamaan
			status, body := call("POST", projectsURL, map[string]any{
				"title":        fmt.Sprintf("Proyek Serentak %02d", n),
				"technologies": []string{"Race"},
				"year":         2026,
			})
			switch {
			case status == 200:
				atomic.AddInt32(&okCount, 1)
				fmt.Printf("[3] serentak %02d: HTTP 200 (masuk, id proyek %s)\n",
					n, lastID(body))
			case status == 409 && body.Code == "PROJECT_LIMIT":
				atomic.AddInt32(&conflictCount, 1)
				fmt.Printf("[3] serentak %02d: HTTP 409 PROJECT_LIMIT (ditolak)\n", n)
			default:
				atomic.AddInt32(&otherCount, 1)
				fmt.Printf("[3] serentak %02d: HTTP %d %s (TIDAK DIHARAPKAN)\n",
					n, status, body.Code)
			}
		}(i)
	}
	close(start)
	wg.Wait()

	// Langkah 4: hitung isi akhir di database.
	status, body = call("GET", portfolioURL, nil)
	final := len(body.Data.Projects)
	fmt.Println("----------------------------------------------------------")
	fmt.Printf("[4] sukses 200      : %d\n", okCount)
	fmt.Printf("[4] ditolak 409     : %d\n", conflictCount)
	fmt.Printf("[4] lainnya         : %d\n", otherCount)
	fmt.Printf("[4] proyek di DB    : %d (GET HTTP %d)\n", final, status)
	if okCount == 2 && conflictCount == 8 && otherCount == 0 && final == 20 {
		fmt.Println("[4] HASIL: BENAR — tepat 2 lolos, batas 20 bertahan.")
	} else {
		fmt.Println("[4] HASIL: SALAH — batas bocor atau respons tidak sesuai.")
	}
}

func lastID(r apiResponse) string {
	if n := len(r.Data.Projects); n > 0 {
		return r.Data.Projects[n-1].ID
	}
	return "?"
}
