package report

import (
	"math"
	"time"
)

const (
	OfcomDailyGBP = 10.34
	ofcomWaitDays = 2
)

type Compensation struct {
	Currency         string  `json:"currency,omitempty"`
	MonthlyPrice     float64 `json:"monthly_price"`
	ProviderDowntime float64 `json:"provider_downtime_seconds"`
	ExcludedSeconds  float64 `json:"excluded_seconds"`
	MonthSeconds     float64 `json:"month_seconds"`
	ProRata          float64 `json:"pro_rata"`
	OfcomDays        int     `json:"ofcom_days,omitempty"`
	OfcomAmountGBP   float64 `json:"ofcom_amount_gbp,omitempty"`
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func Compensate(sum MonthlySummary, monthlyPrice float64, currency, country string, now time.Time) Compensation {
	c := Compensation{Currency: currency, MonthlyPrice: monthlyPrice}
	if start, err := time.ParseInLocation("2006-01", sum.Month, now.Location()); err == nil {
		c.MonthSeconds = start.AddDate(0, 1, 0).Sub(start).Seconds()
	} else {
		c.MonthSeconds = 30 * 86400
	}
	for _, e := range sum.Events {
		if e.Type != "disconnected" {
			continue
		}
		if e.OnBattery || e.AfterGap {
			c.ExcludedSeconds += e.DurationSecs
			continue
		}
		if country == "GB" && e.DurationSecs > ofcomWaitDays*86400 {
			c.OfcomDays += int(math.Ceil((e.DurationSecs - ofcomWaitDays*86400) / 86400))
		}
	}
	c.ProviderDowntime = max(0, sum.TotalDowntimeSecs-c.ExcludedSeconds)
	if monthlyPrice > 0 && c.MonthSeconds > 0 {
		c.ProRata = round2(monthlyPrice * c.ProviderDowntime / c.MonthSeconds)
	}
	if c.OfcomDays > 0 {
		c.OfcomAmountGBP = round2(float64(c.OfcomDays) * OfcomDailyGBP)
	}
	return c
}
