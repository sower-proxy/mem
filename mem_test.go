package mem_test

import (
	"testing"
	"time"

	"github.com/sower-proxy/mem"
)

func Test_Get_Succ(t *testing.T) {
	cache := mem.NewCache(time.Second, func(key string) (string, error) {
		return time.Now().Format(time.StampMilli), nil
	})

	v1, _ := cache.Get("now")
	time.Sleep(600 * time.Millisecond)
	v2, _ := cache.Get("now")
	time.Sleep(600 * time.Millisecond)
	v3, _ := cache.Get("now")

	if v1 != v2 || v2 == v3 {
		t.Error("failed", v1, v2, v3)
	}
}
