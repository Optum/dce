module github.com/Optum/dce

go 1.15

require (
	github.com/360EntSecGroup-Skylar/excelize v1.4.1
	github.com/asaskevich/govalidator v0.0.0-20200108200545-475eaeb16496 // indirect
	github.com/avast/retry-go v2.3.0+incompatible
	github.com/aws/aws-lambda-go v1.11.1
	github.com/aws/aws-sdk-go v1.25.36
	github.com/awslabs/aws-lambda-go-api-proxy v0.5.0
	github.com/caarlos0/env v3.5.0+incompatible
	github.com/go-ozzo/ozzo-validation v3.6.0+incompatible
	github.com/go-sql-driver/mysql v1.4.1 // indirect
	github.com/google/uuid v1.1.1
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/schema v1.1.0
	github.com/gruntwork-io/terratest v0.15.13
	github.com/imdario/mergo v0.3.8
	github.com/mitchellh/mapstructure v1.1.2
	github.com/oleiade/reflections v1.0.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/pquerna/otp v1.2.0 // indirect
	github.com/rebuy-de/aws-nuke v0.0.0-00010101000000-000000000000
	github.com/securego/gosec v0.0.0-20200316084457-7da9f46445fd // indirect
	github.com/stretchr/testify v1.4.0
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	google.golang.org/appengine v1.4.0 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	gopkg.in/oleiade/reflections.v1 v1.0.0
	gopkg.in/yaml.v2 v2.2.8
)

replace github.com/rebuy-de/aws-nuke => github.com/Optum/aws-nuke v1.1.0
