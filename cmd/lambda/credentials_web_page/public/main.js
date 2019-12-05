new Vue({
  el: '#app',
  data:{
      auth: null,
      jwt: "",
      decodedJwt: "",
      encodedCreds: ""
  },
  mounted() {
    this.initCognitoSDK();
    let curUrl = window.location.href;
    this.auth.parseCognitoWebResponse(curUrl);
    this.authenticate()
    this.authorize()
  },
  methods: {
      initCognitoSDK() {
        console.log("Initializing Cognito SDK")
        var authData = {
                AppWebDomain: USER_POOL_APP_WEB_DOMAIN + ".auth." + AWS_CURRENT_REGION + ".amazoncognito.com",
                ClientId: USER_POOL_CLIENT_ID,
                RedirectUriSignIn: "https://" + window.location.hostname + "/" + APIGW_DEPLOYMENT_NAME + "/" + SITE_PATH_PREFIX,
                RedirectUriSignOut: "https://" + window.location.hostname + "/" + APIGW_DEPLOYMENT_NAME + "/" + SITE_PATH_PREFIX,
                TokenScopesArray: ['openid', 'email'],
                UserPoolId: USER_POOL_ID,
            }
        this.auth = new AmazonCognitoIdentity.CognitoAuth(authData);
        var self = this
        this.auth.userhandler = {
          onSuccess: function(result) {
            console.log("Sign in success")
            self.showSignedIn(result)
          },
          onFailure: function(err) {
            console.log("Error!" + err)
          }
        };
        this.auth.useCodeGrantFlow();
      },
      authenticate() {
        console.log("Signing in")
        this.auth.getSession()
      },
      showSignedIn(session) {
          if (session) {
            this.jwt = session.getIdToken().getJwtToken();
            var payload = this.jwt.split('.')[1];
            this.decodedJwt = JSON.parse(atob(payload));
          }
      },
      authorize() {
        // Add the User's Id Token to the Cognito credentials login map.
        AWS.config.update({region:AWS_CURRENT_REGION});
        var logins = {}
        logins[USER_POOL_PROVIDER_NAME] = this.jwt
        AWS.config.credentials = new AWS.CognitoIdentityCredentials({
            IdentityPoolId: IDENTITY_POOL_ID,
            Logins: logins
        });
        var self = this
        AWS.config.credentials.refresh((error) => {
          if (error) {
              self.encodedCreds = ""
              console.error(error);
          } else {
              console.log('Successfully integrated with identity pool!');
              let stsCreds = {}
              stsCreds.accessKeyId = AWS.config.credentials.accessKeyId
              stsCreds.secretAccessKey = AWS.config.credentials.secretAccessKey
              stsCreds.sessionToken = AWS.config.credentials.sessionToken
              stsCreds.expireTime = this.decodedJwt.exp
              self.encodedCreds = btoa(JSON.stringify(stsCreds))
              document.getElementById("credentialscontainer").innerHTML += self.encodedCreds;
          }
        });
      },
      copyToClipboard() {
          var copyTextarea = document.querySelector('.js-copytextarea');
          copyTextarea.focus();
          copyTextarea.select();

          try {
            var successful = document.execCommand('copy');
            var msg = successful ? 'successful' : 'unsuccessful';
            console.log('Copying text command was ' + msg);
          } catch (err) {
            console.log('Oops, unable to copy');
          }
      },
      signOut() {
          this.auth.signOut()
      },
  }
})