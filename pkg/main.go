package main

import (
	"fmt"

	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := dynamodb.New(sess)

	database := db.DB{
		Client:                   svc,
		AccountTableName:         "Accounts",
		LeaseTableName:           "Leases",
		DefaultLeaseLengthInDays: 7,
		ConsistentRead:           false,
	}

	output, err := database.GetLeases(db.GetLeasesInput{
		PrincipalID: "vtulaba1@optumcloud.com",
	}) // Add a string argument here
	if err != nil {
		fmt.Println("err: ", err)
	} else {
		for _, item := range output.Results {
			fmt.Println("item: ", item.AccountID, "PrincipalID: ", item.PrincipalID, "LeaseStatus: ", item.LeaseStatus)
		}
	}

}

//AWS_PROFILE="Contributor-882450339387" go run main.go
