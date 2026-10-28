package main

import (
	"TenzoFlow/internal/config"
	"fmt"
)

func main() {
	cfg := config.MustLoad()
	fmt.Println(cfg)
}
