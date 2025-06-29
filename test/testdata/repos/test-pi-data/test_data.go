package testdata

// This file contains TEST DATA ONLY - no real PI

var testCustomers = []Customer{
	{
		Name:     "John Smith",
		TFN:      "123456782",   // TEST TFN - validated checksum
		ABN:      "51824753556", // TEST ABN
		Medicare: "2123456701",  // TEST Medicare
		BSB:      "062-001",     // TEST BSB
		Account:  "12345678",    // TEST Account
	},
	{
		Name:     "Jane Doe",
		TFN:      "876543210",   // TEST TFN - validated checksum
		ABN:      "88952560394", // TEST ABN
		Medicare: "2234567805",  // TEST Medicare
		BSB:      "123-456",     // TEST BSB
		Account:  "987654321",   // TEST Account
	},
}

type Customer struct {
	Name     string
	TFN      string
	ABN      string
	Medicare string
	BSB      string
	Account  string
	ACN      string
	Passport string
	License  string
}

// Test configuration
const (
	// TEST DATA ONLY
	DEFAULT_TFN   = "123456782"
	DEFAULT_ABN   = "51824753556"
	TEST_MEDICARE = "2123456701"
	TEST_BSB      = "062-001"
	TEST_ACCOUNT  = "12345678"
	TEST_ACN      = "004028077"
	TEST_PASSPORT = "N1234567"
	TEST_LICENSE  = "12345678" // NSW format
)
