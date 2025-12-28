package provider

// common error messages.
const (
	msgErrGetPgConnection         = "Error establishing a PostgreSQL connection."
	msgErrStartPgTransaction      = "Error starting a PostgreSQL transaction."
	msgErrCommitPgTransaction     = "Failed to commit PostgreSQL transaction."
	msgErrorExecutingPgAction     = "Error %s PostgreSQL '%s'."
	msgErrorMissingResId          = "Missing resource ID in the request."
	msgErrorParsingProviderData   = "Error parsing provider data."
	msgErrInvalidProviderData     = "Invalid provider data."
	msgErrorPgObjectNotFund       = "PostgreSQL object '%s' not found."
	msgErrorPgObjectNotFundDetail = "The PostgreSQL '%s' with ID '%s' was not found in the current connection."
)

// common validator messages.
const (
	msgRequiredField             = "This field '%s' is required for the resources '%s'."
	msgInvalidFieldPattern       = "The value received for the field '%s' is invalid."
	msgInvalidFieldPatternDetail = "The expected format for the field '%s' is '%s'."
	msgInvalidType               = "Expected type: '%s', received instead: '%T'."
)

type PostgresqlObjectType int

const (
	PGUserFunction PostgresqlObjectType = iota
	PGRole
	PGEventTrigger
	PGDatabase
	PGSchema
)

func (p PostgresqlObjectType) String() string {
	switch p {
	case PGUserFunction:
		return "User Function"
	case PGRole:
		return "Role"
	case PGEventTrigger:
		return "Event Trigger"
	case PGDatabase:
		return "Database"
	case PGSchema:
		return "Schema"
	default:
		return "unknown"
	}
}

type TerraformAction int

const (
	TFCreateAction TerraformAction = iota
	TFDeleteAction
	TFReadAction
	TFUpdateAction
)

func (t TerraformAction) String() string {
	switch t {
	case TFCreateAction:
		return "Create"
	case TFDeleteAction:
		return "Delete"
	case TFReadAction:
		return "Read"
	case TFUpdateAction:
		return "Update"
	default:
		return "unknown"
	}
}
