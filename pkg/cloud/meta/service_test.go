package meta

import "testing"

func TestAggregatedListSupportsPartialSuccess(t *testing.T) {
	tests := []struct {
		name    string
		options int
		want    bool
	}{
		{name: "supported", options: AggregatedList, want: true},
		{name: "unsupported", options: AggregatedList | NoReturnPartialSuccess, want: false},
		{name: "not aggregated", options: 0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &ServiceInfo{options: tt.options}
			if got := info.AggregatedListSupportsPartialSuccess(); got != tt.want {
				t.Errorf("AggregatedListSupportsPartialSuccess() = %t, want %t", got, tt.want)
			}
		})
	}
}
