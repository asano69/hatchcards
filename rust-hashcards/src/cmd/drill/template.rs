
use maud::DOCTYPE;
use maud::Markup;
use maud::html;

use crate::cmd::drill::katex::KATEX_CSS_URL;
use crate::cmd::drill::katex::KATEX_JS_URL;
use crate::cmd::drill::katex::KATEX_MHCHEM_JS_URL;

const HIGHLIGHT_JS_URL: &str =
    "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js";
const HIGHLIGHT_CSS_URL: &str =
    "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github.min.css";

pub fn page_template(body: Markup) -> Markup {
    html! {
        (DOCTYPE)
        html lang="en" {
            head {
                meta charset="utf-8";
                meta name="viewport" content="width=device-width, initial-scale=1";
                title { "hashcards" }
                link rel="stylesheet" href=(KATEX_CSS_URL);
                link rel="stylesheet" href=(HIGHLIGHT_CSS_URL);
                script defer src=(KATEX_JS_URL) {};
                script defer src=(KATEX_MHCHEM_JS_URL) {};
                script defer src=(HIGHLIGHT_JS_URL) {};
                link rel="stylesheet" href="/style.css";
                style { ".card-content { opacity: 0; }" }
                noscript { style { ".card-content { opacity: 1; }" }}
            }
            body {
                (body)
                script src="/script.js" {};
            }
        }
    }
}
