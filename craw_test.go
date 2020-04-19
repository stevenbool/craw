package craw

import (
	"log"
	"testing"
)

func TestCraw_BaiduWordAndSort(t *testing.T) {
	a := []int{1, 2, 3, 4, 5}
	copy(a[1:], a[2:])
	log.Print(a)
}

//百度移动
