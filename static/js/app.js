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
                document.getElementById("client-suggestions").replaceChildren();
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
            const idInput = document.getElementById("client-id");
            if (idInput) {
                idInput.value = "";
            }
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
