# ATLAN GOOGLE SHEETS CONNECTOR

### DOCS

Atlan Google Sheets Connector is a very simple microservice whose job is simply to subscribe to a message broker(KAFKA in this case) through which it gets the questionnaire data / response as input and writes the data on a google sheet.

HIGH LEVEL OVERVIEW
![architectural diagram](./atlan-architectural%20diagram.png)

### API ENDPOINTS

1. `URL: <base-url>/api/google-sheets/integrate`
   Authenticates user with their google accounts to enable access to google sheets.

2. `URL: <base-url>/api/google-sheets/integrate/callback`
   Callback url to finalize client authentication with google

3. `URL: <base-url>/api/google-sheets/create`
   Creates an integration with google sheets and returns a google sheet url in the response

### NOTE

- If given more time I would implement the following:
  - [ ] Write more tests (Unit and End-to-End).
  - [ ] Build out the major part of the ingestor service as it was hard to actually try out the whole flow of the system.
  - [ ] A more robust error handling and logging mechanism.
  - [ ] Write more code documentation.
  - [ ] Retry mechanism for failed messages
