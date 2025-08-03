#!/bin/bash
  nonce=$(date +%s%N | cut -b1-16)
  payload="/api/v2/auth/r/funding/credits/fUSD${nonce}{}"
  signature=$(echo -n "$payload" | openssl dgst -sha384 -hmac "9d0c9800f8f857f5596c47bdbf62f27fc18b519a357" -binary | xxd -p -c 256)

  curl --request POST \
       --url 'https://api.bitfinex.com/v2/auth/r/funding/credits/fUSD' \
       --header 'accept: application/json' \
       --header 'content-type: application/json' \
       --header "bfx-nonce: ${nonce}" \
       --header 'bfx-apikey: 827b626c99eee86ee9a3416cfd04daabdf4a4cc38dc' \
       --header "bfx-signature: ${signature}" \
       --data '{}'