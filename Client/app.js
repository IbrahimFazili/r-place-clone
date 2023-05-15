const express = require('express')
const app = express()
const port = 3000

app.use('/',express.static('static_files')); // this directory has files to be returned

app.listen(port, () => {
  console.log(`Example app listening on port ${port}`)
})