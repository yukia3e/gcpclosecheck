module nested_patterns

go 1.21

require (
	cloud.google.com/go/spanner v1.45.0
)

replace cloud.google.com/go/spanner => ../fake_spanner