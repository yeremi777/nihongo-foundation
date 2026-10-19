# Items are addressed by level and code

One item is looked up at `/api/<section>/{level}/{code}`, not by its UUID as a lesson is. A code is the item's stable identifier within its level and the only one a quiz question carries, so a client opens the item behind a question without a search; a UUID is not a domain term. Serving both was rejected as two routes for one task.
