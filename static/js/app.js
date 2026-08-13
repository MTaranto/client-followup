(function () {
    "use strict";

    let exactLookupSequence = 0;

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
        if (!modal || modal.dataset.prepared === "true") {
            return;
        }
        modal.dataset.prepared = "true";

        modal.querySelectorAll("[data-phone-input]").forEach(formatPhoneInput);
        const followUpForm = modal.querySelector("#followup-form");
        const followUpFields = followUpForm && followUpForm.querySelector("#followup-fields");
        if (followUpFields) {
            setFollowUpFieldsLocked(followUpForm, followUpFields.disabled);
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

    function normalizeName(value) {
        return value.trim().toLocaleLowerCase("pt-BR").normalize("NFD").replace(/\p{M}/gu, "");
    }

    function removeNameDigits(input) {
        const sanitized = input.value.replace(/\p{N}/gu, "");
        if (sanitized !== input.value) {
            input.value = sanitized;
        }
    }

    function setFollowUpFieldsLocked(form, locked) {
        const fields = form && form.querySelector("#followup-fields");
        const saveButton = form && form.querySelector("#followup-save");
        if (fields) {
            fields.disabled = locked;
        }
        if (saveButton) {
            saveButton.disabled = locked;
        }
        if (form) {
            // The disabled fieldset is the authoritative UI gate: client identity
            // must be resolved before any dependent value can be edited or sent.
            form.dataset.clientResolved = locked ? "false" : "true";
        }
    }

    function setClientResolution(form, value) {
        const resolutionInput = form && form.querySelector("#client-resolution");
        if (resolutionInput) {
            resolutionInput.value = value;
        }
    }

    function clearClientMatches(form) {
        const matches = form && form.querySelector("#client-match-choice");
        if (matches) {
            matches.replaceChildren();
        }
    }

    function clearPhoneDecision(form) {
        const actionInput = form && form.querySelector("#phone-change-action");
        const confirmation = form && form.querySelector("#phone-change-confirmation");
        if (actionInput) {
            actionInput.value = "";
            delete actionInput.dataset.phoneChangeValue;
        }
        if (confirmation) {
            confirmation.replaceChildren();
        }
    }

    function formatPhoneInput(input) {
        const originalValue = input.value;
        const originalCursor = input.selectionStart === null ? originalValue.length : input.selectionStart;
        const digitsBeforeCursor = originalValue.slice(0, originalCursor).replace(/\D/g, "").length;
        const digits = originalValue.replace(/\D/g, "").slice(0, 11);
        let formatted = "";

        if (digits.length > 0) {
            formatted = "(" + digits.slice(0, 2);
        }
        if (digits.length > 2) {
            formatted += ") " + digits.slice(2, 7);
        }
        if (digits.length > 7) {
            formatted += "-" + digits.slice(7);
        }

        input.value = formatted;
        if (input !== document.activeElement || input.selectionStart === null) {
            return;
        }

        let cursor = 0;
        let digitsSeen = 0;
        while (cursor < formatted.length && digitsSeen < digitsBeforeCursor) {
            if (/\d/.test(formatted[cursor])) {
                digitsSeen += 1;
            }
            cursor += 1;
        }
        while (cursor < formatted.length && !/\d/.test(formatted[cursor])) {
            cursor += 1;
        }
        if (digitsBeforeCursor >= digits.length) {
            cursor = formatted.length;
        }
        input.setSelectionRange(cursor, cursor);
    }

    function invalidateClientSelection(nameInput) {
        const form = nameInput.closest("form");
        const idInput = form && form.querySelector("#client-id");
        const contactInput = form && form.querySelector("#client-contact");
        if (!idInput || !contactInput || !idInput.value) {
            return;
        }
        if (normalizeName(nameInput.value) === normalizeName(nameInput.dataset.selectedClientName || "")) {
            return;
        }

        idInput.value = "";
        if (contactInput.dataset.autofilledContact !== undefined &&
                contactInput.value === contactInput.dataset.autofilledContact) {
            contactInput.value = "";
        }
        delete nameInput.dataset.selectedClientName;
        delete contactInput.dataset.autofilledContact;
        clearPhoneDecision(form);
        setClientResolution(form, "");
        setFollowUpFieldsLocked(form, true);
    }

    function selectClient(form, client, preserveManualContact) {
        const idInput = form.querySelector("#client-id");
        const nameInput = form.querySelector("#client-name");
        const contactInput = form.querySelector("#client-contact");
        if (!idInput || !nameInput || !contactInput) {
            return;
        }

        idInput.value = String(client.id);
        nameInput.value = client.name;
        nameInput.dataset.selectedClientName = client.name;
        if (!preserveManualContact || !contactInput.value) {
            contactInput.value = client.contact;
            formatPhoneInput(contactInput);
            contactInput.dataset.autofilledContact = contactInput.value;
        } else {
            delete contactInput.dataset.autofilledContact;
        }
        clearClientMatches(form);
        clearPhoneDecision(form);
        setClientResolution(form, "existing");
        setFollowUpFieldsLocked(form, false);
    }

    function chooseNewHomonymousClient(form) {
        const idInput = form.querySelector("#client-id");
        const nameInput = form.querySelector("#client-name");
        const contactInput = form.querySelector("#client-contact");
        if (!idInput || !nameInput || !contactInput) {
            return;
        }
        idInput.value = "";
        if (contactInput.dataset.autofilledContact !== undefined &&
                contactInput.value === contactInput.dataset.autofilledContact) {
            contactInput.value = "";
        }
        delete nameInput.dataset.selectedClientName;
        delete contactInput.dataset.autofilledContact;
        clearClientMatches(form);
        clearPhoneDecision(form);
        setClientResolution(form, "new_homonym");
        setFollowUpFieldsLocked(form, false);
        contactInput.focus();
    }

    function renderClientMatches(form, clients) {
        const container = form.querySelector("#client-match-choice");
        if (!container) {
            return;
        }

        const panel = document.createElement("div");
        panel.className = "client-match-panel";
        const message = document.createElement("p");
        message.textContent = "Existe mais de uma cliente com este nome. Escolha o telefone correto:";
        panel.appendChild(message);

        clients.forEach(function (client) {
            const button = document.createElement("button");
            button.type = "button";
            button.className = "button secondary client-match-option";
            button.textContent = client.contact || "Cliente sem telefone cadastrado";
            button.dataset.clientId = String(client.id);
            button.dataset.clientName = client.name;
            button.dataset.clientContact = client.contact;
            panel.appendChild(button);
        });

        const newClientButton = document.createElement("button");
        newClientButton.type = "button";
        newClientButton.className = "button ghost client-match-new";
        newClientButton.textContent = "Cadastrar nova cliente com este nome";
        panel.appendChild(newClientButton);
        container.replaceChildren(panel);
    }

    function verifyExactClient(nameInput, focusNext) {
        const form = nameInput.closest("form");
        if (!form || form.id !== "followup-form") {
            return;
        }
        clearClientMatches(form);
        const requestedName = nameInput.value.trim();
        if (!requestedName) {
            setClientResolution(form, "");
            setFollowUpFieldsLocked(form, true);
            return;
        }
        const idInput = form.querySelector("#client-id");
        if (idInput && idInput.value &&
                normalizeName(nameInput.dataset.selectedClientName || "") === normalizeName(requestedName)) {
            setClientResolution(form, "existing");
            setFollowUpFieldsLocked(form, false);
            if (focusNext) {
                form.querySelector("#client-contact").focus();
            }
            return;
        }

        const contactInput = form.querySelector("#client-contact");
        const contactAtRequest = contactInput ? contactInput.value : "";
        setClientResolution(form, "");
        setFollowUpFieldsLocked(form, true);
        nameInput.setAttribute("aria-busy", "true");
        // Only the newest lookup may unlock the form. This prevents a slow response
        // for an older name from bypassing the structural identity gate.
        const requestSequence = ++exactLookupSequence;
        fetch("/clients/exact?client_name=" + encodeURIComponent(requestedName), {
            headers: { "Accept": "application/json" }
        }).then(function (response) {
            if (!response.ok) {
                throw new Error("client lookup failed");
            }
            return response.json();
        }).then(function (payload) {
            if (requestSequence !== exactLookupSequence || !form.isConnected ||
                    normalizeName(nameInput.value) !== normalizeName(requestedName)) {
                return;
            }
            const clients = Array.isArray(payload.clients) ? payload.clients : [];
            if (clients.length === 1) {
                const contactWasEdited = contactInput && contactInput.value !== contactAtRequest;
                selectClient(form, clients[0], contactWasEdited);
                if (focusNext) {
                    contactInput.focus();
                }
            } else if (clients.length > 1) {
                renderClientMatches(form, clients);
                if (focusNext) {
                    const firstChoice = form.querySelector(".client-match-option");
                    if (firstChoice) {
                        firstChoice.focus();
                    }
                }
            } else {
                setClientResolution(form, "new");
                setFollowUpFieldsLocked(form, false);
                if (focusNext) {
                    contactInput.focus();
                }
            }
            nameInput.removeAttribute("aria-busy");
        }).catch(function () {
            if (requestSequence !== exactLookupSequence || !form.isConnected ||
                    normalizeName(nameInput.value) !== normalizeName(requestedName)) {
                return;
            }
            nameInput.removeAttribute("aria-busy");
            setFollowUpFieldsLocked(form, true);
            const message = form.querySelector("#followup-form-message");
            if (message) {
                const error = document.createElement("div");
                error.className = "message error";
                error.textContent = "Não foi possível verificar a cliente agora. Tente novamente antes de continuar.";
                message.replaceChildren(error);
            }
        });
    }

    document.addEventListener("keydown", function (event) {
        if (event.key === "Escape") {
            closeModal();
            return;
        }
        if (event.key === "Tab" && event.target.id === "client-name") {
            const form = event.target.closest("#followup-form");
            if (form && form.dataset.clientResolved !== "true") {
                event.preventDefault();
                verifyExactClient(event.target, true);
            }
        }
    });

    document.addEventListener("click", function (event) {
        if (event.target.classList.contains("modal-overlay") || event.target.closest(".modal-close")) {
            closeModal();
            return;
        }

        const clientMatch = event.target.closest(".client-match-option");
        if (clientMatch) {
            const form = clientMatch.closest("form");
            selectClient(form, {
                id: clientMatch.dataset.clientId,
                name: clientMatch.dataset.clientName,
                contact: clientMatch.dataset.clientContact
            }, false);
            form.querySelector("#client-contact").focus();
            return;
        }

        const newClient = event.target.closest(".client-match-new");
        if (newClient) {
            chooseNewHomonymousClient(newClient.closest("form"));
            return;
        }

        const phoneDecision = event.target.closest("[data-phone-change-action]");
        if (phoneDecision) {
            const form = phoneDecision.closest("form");
            const actionInput = form && form.querySelector("#phone-change-action");
            const contactInput = form && form.querySelector("#client-contact");
            if (actionInput && contactInput) {
                actionInput.value = phoneDecision.dataset.phoneChangeAction;
                actionInput.dataset.phoneChangeValue = contactInput.value;
                phoneDecision.closest(".phone-confirmation").remove();
            }
            return;
        }

        const cancelClientPhoneChange = event.target.closest("[data-cancel-client-phone-change]");
        if (cancelClientPhoneChange) {
            const form = cancelClientPhoneChange.closest("form");
            const phoneInput = form && form.querySelector("[name=contact]");
            const confirmationInput = form && form.querySelector("#client-phone-change-confirmation");
            if (phoneInput) {
                phoneInput.value = phoneInput.dataset.originalPhone || phoneInput.value;
            }
            if (confirmationInput) {
                confirmationInput.value = "";
            }
            cancelClientPhoneChange.closest(".phone-confirmation").remove();
            return;
        }

        const confirmClientPhoneChange = event.target.closest("[data-confirm-client-phone-change]");
        if (confirmClientPhoneChange) {
            const confirmationInput = confirmClientPhoneChange.closest("form").querySelector("#client-phone-change-confirmation");
            if (confirmationInput) {
                confirmationInput.value = "confirm";
            }
        }

        if (event.target.closest("[data-print]")) {
            window.print();
        }
    });

    document.addEventListener("input", function (event) {
        if (event.target.id === "dashboard-client-search" || event.target.id === "client-name" ||
                event.target.matches("[data-client-name-input]")) {
            removeNameDigits(event.target);
        }
        if (event.target.matches("[data-phone-input]")) {
            formatPhoneInput(event.target);
        }
        if (event.target.id === "client-name") {
            invalidateClientSelection(event.target);
            clearClientMatches(event.target.closest("form"));
            setClientResolution(event.target.closest("form"), "");
            setFollowUpFieldsLocked(event.target.closest("form"), true);
        }
        if (event.target.matches("#client-edit-region [name=contact]")) {
            const form = event.target.closest("form");
            const confirmationInput = form && form.querySelector("#client-phone-change-confirmation");
            const message = form && form.querySelector("#client-edit-message");
            if (confirmationInput) {
                confirmationInput.value = "";
            }
            if (message) {
                message.replaceChildren();
            }
        }
        if (event.target.id === "client-contact") {
            const form = event.target.closest("form");
            const actionInput = form && form.querySelector("#phone-change-action");
            if (actionInput && actionInput.value &&
                    event.target.value !== actionInput.dataset.phoneChangeValue) {
                clearPhoneDecision(form);
            } else {
                const confirmation = form && form.querySelector("#phone-change-confirmation");
                if (confirmation) {
                    confirmation.replaceChildren();
                }
            }
            if (event.target.dataset.autofilledContact !== undefined &&
                    event.target.value !== event.target.dataset.autofilledContact) {
                delete event.target.dataset.autofilledContact;
            }
        }
    }, true);

    document.addEventListener("change", function (event) {
        if (event.target.id === "client-name") {
            const form = event.target.closest("#followup-form");
            if (form && form.dataset.clientResolved !== "true") {
                verifyExactClient(event.target, false);
            }
        }
    });

    document.body.addEventListener("closeModal", closeModal);
    document.body.addEventListener("followupSaved", function () {
        const searchInput = document.getElementById("dashboard-client-search");
        const results = document.getElementById("client-search-results");
        if (searchInput) {
            searchInput.value = "";
        }
        if (results) {
            results.replaceChildren();
        }
    });
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
