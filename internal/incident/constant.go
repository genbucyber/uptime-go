package incident

type Severity string
type Status string
type Type string

// Monitor status constants
const (
	StatusUP      = "UP"
	StatusDOWN    = "DOWN"
	StatusPENDING = "PENDING" // Waiting for retry verification
)

const (
	INFO     Severity = "INFO"
	LOW      Severity = "LOW"
	MEDIUM   Severity = "MEDIUM"
	HIGH     Severity = "HIGH"
	CRITICAL Severity = "CRITICAL"
)

const (
	FalsePositive   Status = "False-Positive"
	OnInvestigation Status = "On Investigation"
	Resolved        Status = "Resolved"
)

const (
	UnexpectedStatusCode Type = "unexpected_status_code"
	SSLExpired           Type = "certificate_expired"
	Timeout              Type = "timeout"
	ContentSize			 Type = "content_size"
)

const (
	EventWebsiteDown               string = "website_down"
	EventWebsiteCertificateExpired string = "website_certificate_expired"
)

const (
	MinSampleSizeForAnomal		= 5
	RollingSampeLimit			= 20
	MinAbsoluteChangeBytes		= 50 * 1024
	MinPercentageChange			= 0.70
)
