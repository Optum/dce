## Dcs Reset Generator

This lambda will read from DynamoDB and create SQS entries for the Dcs Nuke function to consume.

events.dyanmoDBEvent{} ---> enqueue() Lambda ---> sqs.SendMessage()

The structure passed to the queue in the message body is the dcs struct as a string:

```golang
var string "aws_account_string"
```

As with any struct, there is not a guarantee of all fields being populated, but a current DB record will look like:

```json
{'aws_account_id'}
```
