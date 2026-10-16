# Troubleshooting

## Replay failed receives

Receive events that still fail after Lambda's retries are kept in a dead-letter
queue for 14 days. Once the cause is fixed, run this from the repository root
to replay them all, printing why each one failed. Replaying is safe to repeat:
a stored email is left unchanged (its notifications are sent again), and a
deleted one is skipped.

```shell
function_name=PROJECT-ENV-email_receive # e.g. mailbox-prod-email_receive
queue=$(terraform output -raw email_receive_dlq_url)
while msg=$(aws sqs receive-message --queue-url "$queue" --wait-time-seconds 5 \
    --query 'Messages[0]' --output json) &&
  [ "${msg:-null}" != null ]; do
  echo "$msg" | jq -r '.Body | fromjson | .responsePayload.errorMessage // .requestContext.condition'
  echo "$msg" | jq -r '.Body | fromjson | .requestPayload' > event.json
  aws lambda invoke --function-name "$function_name" \
    --invocation-type Event --payload fileb://event.json /dev/null &&
    aws sqs delete-message --queue-url "$queue" \
      --receipt-handle "$(echo "$msg" | jq -r .ReceiptHandle)"
done
```
