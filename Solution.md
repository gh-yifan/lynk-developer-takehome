
# storage
3-tier map of priority to FIFO list of command pointers for the complete ordered listing
3-tier map of priority to FIFO list of pending command pointers for dispatch
map of command ID (UUID) to its element in the complete priority list for O(1) lookup (PATCH and DELETE)
map of pending command ID to its element in the pending list for O(1) removal
the priority and pending lists point to the same command object, so status changes stay consistent


# operations
GET /commands - combine the complete lists in HIGH, MEDIUM, LOW order; list order preserves FIFO within each priority
POST /commands - validate, create a PENDING command, append it to both its complete priority list and pending list, and index both elements
POST /commands/dispatch - remove and return the front pending command from the highest-priority non-empty list; set it to EXECUTING and retain it in the complete priority list
PATCH command - look up through the ID map, validate the transition, update status and timestamp; remove it from pending when transitioning PENDING -> EXECUTING
DELETE command - remove it from the complete priority list and ID map; if still PENDING, also remove it from the pending list and pending ID map


# validations
command text must not be blank and must be at most 198 characters
priority and status values must be valid
PATCH only allows PENDING -> EXECUTING and EXECUTING -> COMPLETED|FAILED; completed and failed are terminal
PATCH and DELETE require an existing command ID; malformed UUIDs follow the API's 404 response contract
dispatch returns 404 when no pending command is available

# request validation
request DTOs (Data Transfer Objects) define the JSON fields and validation rules for each endpoint; they are separate from the stored Command model
register the validator library's notblank rule once and declare it on the command DTO alongside the 198-character limit
trim leading and trailing command whitespace before validation and storage; preserve internal whitespace
validate each endpoint's DTO in its handler because POST and PATCH have different fields and rules; no general validation middleware
translate validator failures into concise, stable API messages rather than returning raw library errors

# considerations
direct maps provide O(1) lookup by command ID; linked-list elements allow O(1) removal once located
the complete and pending lists hold pointers to one command object as the single source of truth
mutex protects shared maps, lists, and command state from concurrent HTTP requests
state transitions are checked in PATCH; middleware is unnecessary because validation is endpoint-specific

# decisions
use the validator package for DTO rules and custom notblank behavior; keep API error wording in application code
keep validation endpoint-specific rather than using general middleware
use mux path-value extraction for command IDs
use linked lists because commands can be deleted by ID without shifting remaining elements; pending lists also make dispatch select the next command directly

# tests
dispatch returns 404 for an empty pending queue and selects commands by priority and FIFO order
newly created higher-priority commands dispatch before lower-priority pending commands; newly created lower-priority commands do not overtake higher-priority work
delete at beginning, middle, and end - ensure FIFO order is preserved
GET after PATCH - ensure the updated status and updated_at timestamp are reflected
race-enabled tests cover competing status transitions and concurrent command creation
API regression tests cover command validation boundaries and rejected transitions preserving status

# how to run
deploy server: `make deploy`
test the server: `make test`
demo UI to exercise the server: `make run` 

demo UI is available at localhost:8081/demo
open api spec is available at localhost:8080/docs

[Demo](https://github.com/user-attachments/assets/74ef10ad-6d64-483d-8e04-de08908f9d13)

