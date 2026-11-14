import { jsx } from "@app/html-jsx";
import AppCss from "./app.css";

export function Styles() {
  return (
    <>
      <link rel="preconnect" href="https://fonts.googleapis.com" />
      <link
        rel="preconnect"
        href="https://fonts.gstatic.com"
        crossorigin="anonymous"
      />
      <font-awesome version="6" />
      <AppCss />
    </>
  );
}
