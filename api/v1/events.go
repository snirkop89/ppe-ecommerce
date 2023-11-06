package v1

type OrderReceived struct {
	Header Header `json:"header"`
	Order
}

type OrderPickedAndPacked struct {
	Header Header `json:"header"`
	Order
}
