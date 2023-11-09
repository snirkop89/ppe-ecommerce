package v1

type OrderReceived struct {
	Header Header `json:"header"`
	Order
}

type OrderPickedAndPacked struct {
	Header Header `json:"header"`
	Order
}

type OrderError struct {
	Header Header        `json:"header"`
	Event  OrderReceived `json:"event"`
}

type OrderConfirmed struct {
	Header Header `json:"header"`
	Order
}
