'use strict'
const path = require('path');
const express = require('express')
const awsServerlessExpress = require('aws-serverless-express')
const app = express()

app.set('view engine', 'ejs');

app.get('/', function(req, res) {
    res.redirect("/api/site")
});
app.get('/site', function(req, res) {
    res.render('index', {
        SITE_PATH_PREFIX: process.env.SITE_PATH_PREFIX,
        APIGW_DEPLOYMENT_NAME: process.env.APIGW_DEPLOYMENT_NAME,
        IDENTITY_POOL_ID: process.env.IDENTITY_POOL_ID,
        USER_POOL_PROVIDER_NAME: process.env.USER_POOL_PROVIDER_NAME,
        USER_POOL_CLIENT_ID: process.env.USER_POOL_CLIENT_ID,
        USER_POOL_APP_WEB_DOMAIN: process.env.USER_POOL_APP_WEB_DOMAIN,
        USER_POOL_ID: process.env.USER_POOL_ID
    });
});
app.use('/public', express.static('public'))

const server = awsServerlessExpress.createServer(app)

exports.lambda_handler = (event, context) => { 
    console.log(event)
    awsServerlessExpress.proxy(server, event, context) }