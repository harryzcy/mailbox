# Troubleshooting

## Replay failed receives

Receive events that still fail after Lambda's retries are kept in a dead-letter
queue for 14 days. Once the cause is fixed, run this from the repository root
to replay them in turn, printing why each one failed. It stops at the first
event that fails again, which stays in the queue. Replaying is safe to repeat:
a stored email is left unchanged (its notifications are sent again), and a
deleted one is skipped.

```shell
function_name=$(terraform output -raw email_receive_function_name)
queue=$(terraform output -raw email_receive_dlq_url)
while msg=$(aws sqs receive-message --queue-url "$queue" --wait-time-seconds 5 \
    --query 'Messages[0]' --output json) &&
  [ "${msg:-null}" != null ]; do
  jq -r '.Body | fromjson | .responsePayload.errorMessage // .requestContext.condition' <<<"$msg"
  error=$(aws lambda invoke --function-name "$function_name" \
    --payload fileb:///dev/stdin --query FunctionError --output text /dev/null \
    <<<"$(jq -c '.Body | fromjson | .requestPayload' <<<"$msg")") &&
    [ "$error" = None ] || { echo "replay failed, stopping"; break; }
  aws sqs delete-message --queue-url "$queue" \
    --receipt-handle "$(jq -r .ReceiptHandle <<<"$msg")"
done
```
