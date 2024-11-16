# Running the application

1. You'll need to configure OAuth 2.0 project/application in Google Cloud
   - Permissions required "https://www.googleapis.com/auth/drive.readonly", "https://www.googleapis.com/auth/userinfo.profile"
2. Add file `.env` and follow the example located at `.env.example`
3. To start the application `go run .`
4. Access http://localhost:8080
5. Click in "Login with Google"
6. You will be redirected to Google's OAuth 2.0 page
