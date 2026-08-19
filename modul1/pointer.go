package main

import "fmt"

// Menukar nilai dua integer di alamat memori aslinya
func swap(a, b *int) {
	temp := *a
	*a = *b
	*b = temp
}

// Menambahkan item ke slice asli melalui pointer slice
func updateSlice(s *[]string, newItem string) {
	*s = append(*s, newItem)
}

func Pointer() {

	// test swap
	fmt.Printf("=== Test Swap ===\n\n")
	x, y := 10, 20

	fmt.Printf("[before] x = %d (address: %p), y = %d (address: %p)\n", x, &x, y, &y)
	swap(&x, &y)

	fmt.Printf("[after] x = %d (address: %p), y = %d (address: %p)\n\n", x, &x, y, &y)

	// test updateSlice
	fmt.Printf("=== Test updateSlice ===\n\n")
	fruits := []string{"Apel", "Mangga"}

	fmt.Printf("[before] fruits = %v (len: %d, cap: %d)\n", fruits, len(fruits), cap(fruits))

	updateSlice(&fruits, "Jeruk")
	updateSlice(&fruits, "Pisang")

	fmt.Printf("[after] fruits = %v (len: %d, cap: %d)\n", fruits, len(fruits), cap(fruits))

	fmt.Printf("\n")

}
