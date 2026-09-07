package normalize

import "math"

// JapanConsumptionTax is the rate included in Japanese consumer prices. Japan's 総額表示義務
// (total-price display rule, mandatory since April 2021) requires consumer-facing prices to be
// shown tax-included, which is why every Japanese source in this project lists 税込 prices.
const JapanConsumptionTax = 0.10

// PriceExcludingTax converts a tax-inclusive price to the tax-free (税抜 / TAXFREE) price that an
// exporting buyer pays.
//
// The result is rounded UP to the next whole unit. That is not a guess: Jackroad publishes both
// figures on its product pages, and across 14 sampled products ceil matched the published TAXFREE
// value 14/14 while round-half matched only 7/14 (e.g. ¥498,000 incl. tax is published as
// ¥452,728, not the ¥452,727 that rounding gives).
func PriceExcludingTax(inclusive, rate float64) float64 {
	if inclusive <= 0 || rate <= 0 {
		return inclusive
	}
	return math.Ceil(inclusive / (1 + rate))
}
