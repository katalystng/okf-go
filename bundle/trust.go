package bundle

import (
	"fmt"
	"strings"
)

// The trust tiers of OKF v0.2 section 5.3, lowest to highest.
const (
	TierUnverified       = "unverified"
	TierMachineConfirmed = "machine-confirmed"
	TierHumanReviewed    = "human-reviewed"
)

// Status returns the concept's lifecycle status, per OKF v0.2 section 5.4. An absent status reads as
// stable.
func (d *Document) Status() string {
	if s := d.String("status"); s != "" {
		return s
	}
	return "stable"
}

// TrustTier derives the concept's trust tier from verified, per OKF v0.2 section 5.3.
func (d *Document) TrustTier() string {
	events := d.Maps("verified")
	if len(events) == 0 {
		return TierUnverified
	}
	for _, e := range events {
		if strings.HasPrefix(fmt.Sprint(e["by"]), "human:") {
			return TierHumanReviewed
		}
	}
	return TierMachineConfirmed
}
