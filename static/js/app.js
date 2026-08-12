(function () {
    "use strict";

    function modalContainer() {
        return document.getElementById("modal-container");
    }

    function closeModal() {
        const container = modalContainer();
        if (container) {
            container.replaceChildren();
        }
        document.body.classList.remove("modal-open");
    }

    function prepareModal() {
        const container = modalContainer();
        const modal = container && container.querySelector(".modal-overlay");
        document.body.classList.toggle("modal-open", Boolean(modal));
        if (!modal) {
            return;
        }

        const autofocus = modal.querySelector("[autofocus]");
        if (autofocus) {
            autofocus.focus();
        }
        const highlighted = modal.querySelector(".highlighted");
        if (highlighted) {
            highlighted.scrollIntoView({ block: "center", behavior: "smooth" });
        }
    }

    function clearSuggestions() {
        const suggestions = document.getElementById("client-suggestions");
        if (suggestions) {
            suggestions.replaceChildren();
        }
    }

    function invalidateClientSelection(nameInput) {
        const idInput = document.getElementById("client-id");
        const contactInput = document.getElementById("client-contact");
        if (!idInput || !contactInput || !idInput.value) {
            return;
        }
        if (nameInput.value === nameInput.dataset.selectedClientName) {
            return;
        }

        idInput.value = "";
        if (contactInput.dataset.autofilledContact !== undefined &&
                contactInput.value === contactInput.dataset.autofilledContact) {
            contactInput.value = "";
        }
        delete nameInput.dataset.selectedClientName;
        delete contactInput.dataset.autofilledContact;
    }

    document.addEventListener("keydown", function (event) {
        if (event.key === "Escape") {
            closeModal();
        }
    });

    document.addEventListener("click", function (event) {
        if (event.target.classList.contains("modal-overlay") || event.target.closest(".modal-close")) {
            closeModal();
            return;
        }

        const suggestion = event.target.closest(".suggestion");
        if (suggestion) {
            const idInput = document.getElementById("client-id");
            const nameInput = document.getElementById("client-name");
            const contactInput = document.getElementById("client-contact");
            if (idInput && nameInput && contactInput) {
                idInput.value = suggestion.dataset.clientId;
                nameInput.value = suggestion.dataset.clientName;
                contactInput.value = suggestion.dataset.clientContact;
                nameInput.dataset.selectedClientName = suggestion.dataset.clientName;
                contactInput.dataset.autofilledContact = suggestion.dataset.clientContact;
                clearSuggestions();
                contactInput.focus();
            }
            return;
        }

        if (event.target.closest("[data-print]")) {
            window.print();
        }
    });

    document.addEventListener("input", function (event) {
        if (event.target.id === "client-name") {
            invalidateClientSelection(event.target);
            if (!event.target.value.trim()) {
                clearSuggestions();
            }
        }
        if (event.target.id === "client-contact" &&
                event.target.dataset.autofilledContact !== undefined &&
                event.target.value !== event.target.dataset.autofilledContact) {
            delete event.target.dataset.autofilledContact;
        }
    });

    document.addEventListener("htmx:beforeRequest", function (event) {
        if (event.detail.elt.id === "client-name" && !event.detail.elt.value.trim()) {
            event.preventDefault();
            clearSuggestions();
        }
    });

    document.body.addEventListener("closeModal", closeModal);
    document.body.addEventListener("htmx:afterSwap", prepareModal);
    document.body.addEventListener("htmx:beforeSwap", function (event) {
        const status = event.detail.xhr.status;
        if (status >= 400 && status < 500) {
            event.detail.shouldSwap = true;
            event.detail.isError = false;
        }
    });
    document.body.addEventListener("htmx:responseError", function () {
        const flash = document.getElementById("flash");
        if (flash && !flash.textContent.trim()) {
            const message = document.createElement("div");
            message.className = "message error";
            message.textContent = "A operação não pôde ser concluída. Tente novamente.";
            flash.replaceChildren(message);
        }
    });

    const container = modalContainer();
    if (container) {
        new MutationObserver(prepareModal).observe(container, { childList: true, subtree: true });
    }
}());
