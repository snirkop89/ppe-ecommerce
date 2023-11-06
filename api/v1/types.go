package v1

import (
	"time"

	"github.com/google/uuid"
	"github.com/snirkop89/ppe-ecommerce/validator"
)

type Header struct {
	ID          string    `json:"id"`          // GUID representing the event
	PublishedAt time.Time `json:"publishedAt"` // Time when event was published
}

type Order struct {
	OrderID  string    `json:"orderId"`
	Products []Product `json:"products"`
	Customer Customer  `json:"customer"`
}

func ValidateOrder(v *validator.Validator, order *Order) {
	v.Check(len(order.Products) > 0, "products", "must contain at least 1 product")

	for _, p := range order.Products {
		v.Check(p.Quantity > 0, "products", "quantity must be more than zero")
	}

	v.Check(validator.Matches(order.Customer.Email, validator.EmailRX), "email", "invalid customer email address")
	v.Check(len(order.Customer.FirstName) > 1, "firstName", "must be more than 2 characters")
	v.Check(len(order.Customer.LastName) > 2, "lastName", "must be more than 2 characters")
}

func (o Order) ToOrderReceivedEvent() OrderReceived {
	return OrderReceived{
		Header: Header{
			ID:          uuid.NewString(),
			PublishedAt: time.Now(),
		},
		Order: o,
	}
}

type Product struct {
	ProductID string `json:"productId"`
	// Quantity of the product. Can represent the stoage of the amount ordered.
	Quantity int `json:"quantity"`
}

type Customer struct {
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	Email           string `json:"emailAddress"`
	ShippingAddress struct {
		Street     string `json:"street"`
		City       string `json:"city"`
		State      string `json:"state"`
		PostalCode string `json:"postalCode"`
	} `json:"shippingAddress"`
}
