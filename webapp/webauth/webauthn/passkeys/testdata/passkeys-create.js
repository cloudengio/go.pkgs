let randomNumber = Math.floor(Math.random() * 1000);
let username = `Test User ${randomNumber}`;
let email = `test${randomNumber}@example.com`;
console.log(`Creating passkey for ${username} with email ${email}`);
createPasskey(email, username);
console.log(`Created passkey for ${username} with email ${email}`);
