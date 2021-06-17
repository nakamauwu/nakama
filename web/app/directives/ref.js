import { directive } from "lit-html";

export const ref = directive(refInstance => part => {
    refInstance.current = part.committer.element;
})
