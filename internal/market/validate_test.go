package market

import (
	"testing"
	"time"
)

func TestMarketData_Validate(t *testing.T) {
	tests := []struct {
		name    string
		data    MarketData
		wantErr bool
	}{
		{
			name: "valid complete data",
			data: MarketData{
				Symbol:    "BTC",
				PriceUSD:  "65000.50",
				MarketCap: "1300000000000",
				Volume24h: "50000000000",
				Change24h: "2.5",
				Provider:  "coingecko",
				FetchedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "valid minimal data",
			data: MarketData{
				Symbol:   "ETH",
				PriceUSD: "3500.25",
				Provider: "coingecko",
			},
			wantErr: false,
		},
		{
			name: "missing symbol",
			data: MarketData{
				PriceUSD: "100",
				Provider: "coingecko",
			},
			wantErr: true,
		},
		{
			name: "missing price",
			data: MarketData{
				Symbol:   "BTC",
				Provider: "coingecko",
			},
			wantErr: true,
		},
		{
			name: "invalid price",
			data: MarketData{
				Symbol:   "BTC",
				PriceUSD: "not-a-number",
				Provider: "coingecko",
			},
			wantErr: true,
		},
		{
			name: "missing provider",
			data: MarketData{
				Symbol:   "BTC",
				PriceUSD: "65000",
			},
			wantErr: true,
		},
		{
			name: "invalid market cap",
			data: MarketData{
				Symbol:    "BTC",
				PriceUSD:  "65000",
				MarketCap: "invalid",
				Provider:  "coingecko",
			},
			wantErr: true,
		},
		{
			name: "invalid volume",
			data: MarketData{
				Symbol:    "BTC",
				PriceUSD:  "65000",
				Volume24h: "bad",
				Provider:  "coingecko",
			},
			wantErr: true,
		},
		{
			name: "invalid change",
			data: MarketData{
				Symbol:    "BTC",
				PriceUSD:  "65000",
				Change24h: "xyz",
				Provider:  "coingecko",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.data.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
