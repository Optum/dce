## Redbox Reset Generator

Reference for the workflow in this lambda:
-- https://www.lucidchart.com/documents/view/844c2afe-2e6d-4ba3-8139-96cb16e79d7c

This lambda will read from DynamoDB and create SQS entries for the Redbox Nuke function to consume.

events.dyanmoDBEvent{} ---> enqueue() Lambda ---> sqs.SendMessage()

The structure passed to the queue in the message body is the redBox struct as a string:

```golang
var string "aws_account_string"
```

As with any struct, there is not a guarantee of all fields being populated, but a current DB record will look like:

```json
{'aws_account_id'}
```
