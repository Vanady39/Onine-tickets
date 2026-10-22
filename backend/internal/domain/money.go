package domain

import "fmt"

// Money is an amount in kopecks (1/100 RUB). Integer arithmetic avoids float rounding.
type Money int64

// RUB converts whole roubles to Money.
func RUB(roubles int64) Money { return Money(roubles * 100) }

func (m Money) String() string {
	return fmt.Sprintf("%d.%02d RUB", int64(m)/100, int64(m)%100)
}
