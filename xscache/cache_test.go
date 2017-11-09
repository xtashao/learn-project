package xscache

import (
	"reflect"
	"testing"
)

func TestCache(t *testing.T) {
	type args struct {
		table string
	}
	tests := []struct {
		name string
		args args
		want *CacheTable
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Cache(tt.args.table); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Cache() = %v, want %v", got, tt.want)
			}
		})
	}
}
