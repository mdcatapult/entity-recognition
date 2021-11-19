# Recognition API

## `/recognisers`
### `GET`
**Lists available recognisers.**


## `/text`
### `POST`
**Converts html to text.**

#### Request body
Any valid html.

#### Headers
* A content type header set to `text/html` must be included.


## `/tokens`
### `POST`
**Extracts whitespace delimited tokens.**

#### Request body
Any raw text or valid html.

#### Headers
* A content type header set to `text/html` or `text/plain` must be included.

## `/entities`
### `POST`
**Attempts to recognise and resolve entities in the request body**

#### Request body
Any raw text or valid html.

#### Query parameters
* `allRecognisers=true`: Uses all available recognisers for entity recognition and resolution.
* `recogniser=<recogniser-name>`: Uses the specific downstream recogniser for entity recognition and resolution.
Multiple recognisers can be set by setting the same query parameters multiple times. **At least one recogniser must be provided**.

#### Headers
* A content type header set to `text/html` or `text/plain` must be included.
* Optional headers can be added corresponding to each requested recogniser.
This allows the caller to modify the proxied request to the downstream recogniser.
Currently only the addition of query parameters is supported.
The header names should be in the format `x-recogniser-name`.
The value of the header should be base64 encoded json with the following format: 
    ```json
    {
      "queryParameters": {
        "key": ["value"]
      }
    }
    ```
