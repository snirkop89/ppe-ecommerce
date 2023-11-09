package v1

import (
	"time"

	"github.com/google/uuid"
	"github.com/snirkop89/ppe-ecommerce/core/validator"
)

type Header struct {
	ID          string    `json:"id"`          // GUID representing the event
	PublishedAt time.Time `json:"publishedAt"` // Time when event was published
}

func NewHeader() Header {
	return Header{
		ID:          uuid.NewString(),
		PublishedAt: time.Now(),
	}
}

type Order struct {
	OrderID  string    `json:"orderId"`
	Products []Product `json:"products"`
	Customer Customer  `json:"customer"`
}

func ValidateOrder(v *validator.Validator, order *Order) {
	v.Check(len(order.Products) > 0, "products", "must contain at least 1 product")

	for _, p := range order.Products {
		_, err := uuid.Parse(p.ProductID)
		v.Check(err == nil, "productId", "productId is not valid")
		v.Check(p.Quantity > 0, "quantity", "quantity must be more than zero")
	}

	v.Check(validator.Matches(order.Customer.Email, validator.EmailRX), "email", "invalid customer email address")
	v.Check(len(order.Customer.FirstName) > 1, "firstName", "must be more than 2 characters")
	v.Check(len(order.Customer.LastName) > 2, "lastName", "must be more than 2 characters")

	v.Check(len(order.Customer.ShippingAddress.City) > 0, "city", "is required")
	v.Check(len(order.Customer.ShippingAddress.PostalCode) > 0, "postalCode", "is required")
	v.Check(len(order.Customer.ShippingAddress.State) > 0, "state", "is required")
	v.Check(len(order.Customer.ShippingAddress.Street) > 0, "street", "is required")

}

func (o Order) ToOrderReceivedEvent() OrderReceived {
	return OrderReceived{
		Header: NewHeader(),
		Order:  o,
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
