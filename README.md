# apiv2
API v2 (name might change)

## Layout: _(to be cleaned up)_

* `:userID` must be included in all paths for proper and cleaner auth:

```
POST           /api/v2/[path...]/:userID
GET/PUT/DELETE /api/v2/[path...]/:userID/:resID
// harder to achive but something to aim for
GET/PUT/DELETE /api/v2/batch/:userID?ep=path...&id=1&id=2
```

* unified return value, everything returned is json, probably something like:
```
interface BatchResponse {
  {[id: resID]: Response};
}

interface Response {
  code: number; // http status code, if it's >= 400 then it's an error
  data: any; // the returned data, if any
  error: {
    msg?: string; // error message, optional if fields is not null
    fields?: fieldError[]; // per-field error, optional.
  }
}

interface fieldError {
  name: string; // required, field name
  msg: string; // required
}
```
