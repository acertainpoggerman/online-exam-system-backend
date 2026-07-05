# Tomorrow's Plan ("Application Flow")

- [X] (Login/Register) Examiner Acc 
- [X] Create Script with 2 Questions (Short Text & Multiple Choice)
  - [X] Add 2 Questions
  - [X] Update the second Question
  - [X] Duplicate a question twice and delete the 2 questions
- [X] (Login/Register) Examinee Acc
- [X] Create Submission with 2 Answers (For the 2 Questions)
- [X] Change both Answers
- [X] Delete both Answers (Set them as blank lists)

# Today's Plan ("Application Flow")

## Examiner PTI

- [X] (Register & Login) Examiner Account
- [X] Perform all actions for `/scripts` (POST, GET, GET /{id}, PATCH)

## Examiner PTII

- [X] Perform all actions for `/sessions` (POST, GET, GET /{id}, PUT)

## Examinee PTI

- [X] (Register & Login) Examinee Account
- [X] Perform all actions for `/scripts`
  - [X] (Actions to fail) -> POST, GET, PATCH
  - [X] (Actions to succeed) -> GET /{id} (NOTE: Only allow this with ExtSessionService existing if the user is in a started/running session w/ it)
- [X] Perform all actions for `/submissions`


`GET /scripts?search=abc&cursor=XXX&size=10`
size -> default 10, max 20
cursor -> { ts: XX-XX-XXXX XX:XX:XX, uuid: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX }

```sql
SELECT * FROM scripts
WHERE
    scripts.creator_id = @examiner_id::UUID
    AND scripts.title ILIKE '%' || @search || '%' 
    AND (scripts.last_modified_at, scripts.id) < (@cursor_ts::TIMESTAMPTZ, @cursor_id::UUID)
ORDER BY last_modified_at DESC, id DESC
LIMIT @size;
```

"showing X of TOTAL results"

-------------------------------------------------------------------------------

- [X] COMPLETE PAGINATION (Next & Prev [BOTTOM-RIGHT])
- [X] SEARCH
- [X] SCRIPT MENU OPTIONS (Delete)
- [X] IMPLEMENT NEW DROPDOWN (using popover)
- [X] NAV
- [X] REQUEST HOOK
- [ ] PROFILE OPTIONS (Logout, Change Account)
- [ ] MAKE NAV SHOW CORRECT ROUTE IN MIDDLE
- [ ] SCRIPT-WIDE LOCKING & LOGICAL DELETION (Locking vs. Edit Icon on Home Page)
- [ ] SESSIONS
- [ ] SCRIPT MENU OPTIONS (Duplicate/Export/Preview) [In /scripts and /scripts/:id]



current [X]
prev

2 pages total : [0, 1]

load page -> [1]: 0
go to next page -> [1]: 1
go to prev page -> [1]: 0

3 pages total : [0, 1, 2]
load page -> [1]: 0
go to next page -> [1, 2]: 1
go to next page -> [1, 2]: 2
go to prev page -> 

prev, current, next
`[] : cursor1`
`[cursor1] : null`

`[] : cursor1`
`[cursor1] : cursor2`
`[cursor1, cursor2] : null`

if you go to a prev page, pop the last one and instead put the next given by data.next

1 -> onload
2 -> we get this
3 
4
5
6
