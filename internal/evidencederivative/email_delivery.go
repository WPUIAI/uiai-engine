package evidencederivative

const EmailDeliveryPolicySchema = "uiai.evidence_email_delivery_policy.v1"
const MaxEmailMessageBytes = 25 * 1024 * 1024

type EmailTransport string

const (
	EmailTransportSMTP        EmailTransport = "smtp"
	EmailTransportProviderAPI EmailTransport = "provider_api"
)

type EmailDeliveryPolicy struct {
	Schema                  string         `json:"schema"`
	PolicyRef               string         `json:"policy_ref"`
	Transport               EmailTransport `json:"transport"`
	TLSRequired             bool           `json:"tls_required"`
	RecipientConsentRef     string         `json:"recipient_consent_ref"`
	SuppressionEvidenceRef  string         `json:"suppression_evidence_ref"`
	DKIMEvidenceRef         string         `json:"dkim_evidence_ref"`
	SPFEvidenceRef          string         `json:"spf_evidence_ref"`
	DMARCEvidenceRef        string         `json:"dmarc_evidence_ref"`
	NoTracking              bool           `json:"no_tracking"`
	MaxMessageBytes         uint64         `json:"max_message_bytes"`
	BounceReconciliationRef string         `json:"bounce_reconciliation_ref"`
}

func DigestEmailDeliveryPolicy(policy EmailDeliveryPolicy) (string, error) {
	return digest(policy)
}

func ValidateEmailDeliveryPolicy(policy EmailDeliveryPolicy) error {
	if policy.Schema != EmailDeliveryPolicySchema || blank(policy.PolicyRef) ||
		(policy.Transport != EmailTransportSMTP && policy.Transport != EmailTransportProviderAPI) ||
		!policy.TLSRequired || blank(policy.RecipientConsentRef) || blank(policy.SuppressionEvidenceRef) ||
		blank(policy.DKIMEvidenceRef) || blank(policy.SPFEvidenceRef) || blank(policy.DMARCEvidenceRef) ||
		!policy.NoTracking || policy.MaxMessageBytes == 0 ||
		policy.MaxMessageBytes > MaxEmailMessageBytes || blank(policy.BounceReconciliationRef) {
		return ErrDerivativeContractInvalid
	}
	return sizeOK(policy)
}

func ValidateEmailDeliveryReceipt(receipt DeliveryReceipt, policy EmailDeliveryPolicy) error {
	if err := ValidateEmailDeliveryPolicy(policy); err != nil {
		return err
	}
	digest, err := DigestEmailDeliveryPolicy(policy)
	if err != nil {
		return err
	}
	if receipt.PolicyRef != policy.PolicyRef || receipt.PolicySHA256 != digest {
		return ErrDerivativeIdentityMismatch
	}
	return ValidateDelivery(receipt)
}
