package incident

type Severity string
type Status string
type Type string

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
)
