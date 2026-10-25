
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

# considerations
direct map for O(1) PATCH lookup 
use pointers in the direct map and the priority lists to keep a single-source-of-truth (in the heap)
racing requests - to prevent net/http from race, add mutex / synchronization.
state transition validation - PATCH and DELETE; lookup valid state(s) in map; no middleware because it would not be relevant for GET and CREATE
returning a 400 BadRequest for a malformed uuid vs 404. makes more sense a invalid uuid is a input error but to match the contract provided, it must return the 404 response.

# decisions
delegate validation to validate package, bubble validate error up in JSON, minimize custom code & intermixing with business logic
use mux for path parameter extraction (e.g. id)
doubly linked list among commands for O(1) deletion

# new tests
Delete at beginning, middle, and end - ensure FIFO order is preserved
GET after PATCH - ensure the updated at timestamp is modified on PATCH (not stale)
race-enabled test ensure writers don't overwrite status of command
concurrent creates write to the Go map (append to the same slice), potentially losing an entry