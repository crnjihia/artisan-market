module github.com/yourorg/artisan-market/tests

go 1.24.0

require (
	github.com/yourorg/artisan-market/proto v0.0.0
	google.golang.org/protobuf v1.34.2
)

replace github.com/yourorg/artisan-market/proto => ../proto
