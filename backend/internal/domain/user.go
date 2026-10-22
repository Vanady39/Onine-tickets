package domain

import "time"

type Role string

const (
	RoleCustomer Role = "customer"
	RoleCashier  Role = "cashier"
	RoleManager  Role = "manager"
	RoleAdmin    Role = "admin"
)

func (r Role) Valid() bool {
	switch r {
	case RoleCustomer, RoleCashier, RoleManager, RoleAdmin:
		return true
	}
	return false
}

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         Role      `json:"role"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Profile is a user as shown to themselves, with loyalty data for customers.
type Profile struct {
	User
	PaidOrders int  `json:"paidOrders"`
	Loyal      bool `json:"loyal"`
}
