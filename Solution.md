
# storage
3-tier map of priority to list of pointer of commands
map of command id (str) to command pointer for O(1) lookup (PATCH and DELETE)


# operations
/GET commands - stack them together by priority
/POST - create entry in direct map; append to priority list to maintain ordering
/PATCH command - update entry in direct map; pointer not affected
/DELETE command - delete entry in direct map; search in priority list for pointer


# validations
/DELETE - command id exists

# request validation
request DTOs (Data Transfer Objects) define the JSON fields and validation rules for each endpoint; they are separate from the stored Command model
register the validator library's notblank rule once and declare it on the command DTO alongside the 198-character limit
trim leading and trailing command whitespace before validation and storage; preserve internal whitespace
validate each endpoint's DTO in its handler because POST and PATCH have different fields and rules; no general validation middleware
translate validator failures into concise, stable API messages rather than returning raw library errors

# considerations
direct map for O(1) PATCH lookup 
use pointers in the direct map and the priority lists to keep a single-source-of-truth (in the heap)
racing requests - to prevent net/http from race, add mutex / synchronization.
state transition validation - PATCH and DELETE; lookup valid state(s) in map; no middleware because it would not be relevant for GET and CREATE
returning a 400 BadRequest for a malformed uuid vs 404. makes more sense a invalid uuid is a input error but to match the contract provided, it must return the 404 response.

# decisions
use the validator package for DTO rules and custom notblank behavior; keep API error wording in application code
keep validation endpoint-specific rather than using general middleware
use mux for path parameter extraction (e.g. id)
linear search and shift-on-delete

# new tests
Delete at beginning, middle, and end - ensure FIFO order is preserved
GET after PATCH - ensure the updated at timestamp is modified on PATCH (not stale)
race-enabled test ensure writers don't overwrite status of command
concurrent creates write to the Go map (append to the same slice), potentially losing an entry