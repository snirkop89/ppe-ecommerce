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
	Header Header `json:"header"`
	Event  any    `json:"event"`
}

type OrderConfirmed struct {
	Header Header `json:"header"`
	Order
}

type Notification struct {
	Header    Header `json:"header"`
	Type      string `json:"type"`
	Recipient string `json:"recipient"`
	From      string `json:"from"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
}
