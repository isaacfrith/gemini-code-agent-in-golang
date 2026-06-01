
function rot13(str) {
  return str.replace(/[a-zA-Z]/g, function(c) {
    return String.fromCharCode(
      c.charCodeAt(0) + (c.toLowerCase() <= "m" ? 13 : -13)
    );
  });
}

const encodedString = "Pbatenghyngvbaf ba ohvyqvat n pbqr-rqvgvat ntrag!";
const decodedString = rot13(encodedString);
console.log(decodedString);
