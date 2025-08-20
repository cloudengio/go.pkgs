let randomNumber = Math.floor(Crypto.random() * 1000);
let username = `Test User ${randomNumber}`;
let email = `test${randomNumber}@example.com`;
console.log(`Creating passkey for ${username} with email ${email}`);
const pki = createPasskey(email, username);
console.log(`Created passkey for ${username} with email ${email}: ${pki}`);
