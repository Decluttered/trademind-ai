# Customer Service Message Sync UX (Phase F4)

## Page

`/customer/message-sync-tasks`

## States

Consistent with order sync: `pending` / `running` / `success` / `partial_success` / `failed` / `cancelled` / `blocked`

## partial_success

Shows the number of successful/failed conversations, the failure cursor, error message, and whether a retry is possible; not displayed as fully successful or fully failed.

## Unauthorized

When a shop is not authorized, task creation/execution is blocked, with the copy "Pending authorization"; this is not logged as a system bug.

## Failed Tasks

Sync failures go to the failed-task center under `customer_message_sync`; the detail deep link is `/customer/message-sync-tasks?id=`
