
# storage
3-tier map of priority to list of pointer of commands to avoid duplicate storage
map of command id (str) to command for direct lookup (PATCH and DELETE)


# operations
/GET commands - stack them together by priority
/POST - create entry in direct map; append to priority list to maintain ordering
/PATCH command - update entry in direct map; pointer not affected
/DELETE command - delete entry in direct map; search in priority list for pointer


# validations
/DELETE - command id exists

# considerations
use pointers to Commands in the priority list. When Go re-allocates array, it is less heavy to copy pointers than the Command objects.
racing requests - mutex / lock
state transition validation - PATCH and DELETE; lookup valid state(s) in map; no middleware because it would not be relevant for GET and CREATE
returning a 400 BadRequest for a malformed uuid vs 404. makes more sense a invalid uuid is a input error but to match the contract provided, it must return the 404 response.

# decisions
delegate validation to package-level, simple enough to handle validation with validator package, minimize custom code amount
use mux for path parameter extraction (id)
