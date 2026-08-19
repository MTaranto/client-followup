(function () {
    "use strict";

    let exactLookupSequence = 0;
    let phoneLookupSequence = 0;
    let originalDocumentTitle = document.title;

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

    function closeConfirmModal() {
        const confirmOverlay = document.querySelector(".confirm-modal-overlay");
        if (confirmOverlay) {
            confirmOverlay.remove();
            const remainingModal = document.querySelector(".modal-overlay");
            document.body.classList.toggle("modal-open", Boolean(remainingModal));
            return true;
        }
        return false;
    }

    function findDateHiddenInput(visibleInput) {
        if (!visibleInput) return null;
        if (visibleInput.dataset.dateTarget) {
            const target = document.getElementById(visibleInput.dataset.dateTarget);
            if (target) return target;
        }
        return visibleInput.parentElement ? visibleInput.parentElement.querySelector("input[type=hidden]") : null;
    }

    function isLeapYear(year) {
        return (year % 4 === 0 && year % 100 !== 0) || (year % 400 === 0);
    }

    function getDaysInMonth(month, year) {
        const days = [0, 31, isLeapYear(year) ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
        return days[month] || 0;
    }

    function parseBrazilianDate(value) {
        const digits = value.replace(/\D/g, "");
        if (digits.length !== 8) {
            return { valid: false, incomplete: digits.length > 0 };
        }
        const day = parseInt(digits.slice(0, 2), 10);
        const month = parseInt(digits.slice(2, 4), 10);
        const year = parseInt(digits.slice(4, 8), 10);

        if (year < 1000 || year > 9999 || month < 1 || month > 12) {
            return { valid: false, incomplete: false };
        }
        const maxDays = getDaysInMonth(month, year);
        if (day < 1 || day > maxDays) {
            return { valid: false, incomplete: false };
        }
        const iso = `${year}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
        return { valid: true, iso: iso };
    }

    function formatDateInput(input) {
        const originalValue = input.value;
        const originalCursor = input.selectionStart === null ? originalValue.length : input.selectionStart;
        const digitsBeforeCursor = originalValue.slice(0, originalCursor).replace(/\D/g, "").length;
        const digits = originalValue.replace(/\D/g, "").slice(0, 8);
        let formatted = "";

        if (digits.length > 0) {
            formatted = digits.slice(0, 2);
        }
        if (digits.length > 2) {
            formatted += "/" + digits.slice(2, 4);
        }
        if (digits.length > 4) {
            formatted += "/" + digits.slice(4, 8);
        }

        input.value = formatted;

        if (input === document.activeElement && input.selectionStart !== null) {
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
    }

    function initDateInput(visibleInput) {
        const hidden = findDateHiddenInput(visibleInput);
        if (!hidden) return;
        if (hidden.value && /^\d{4}-\d{2}-\d{2}$/.test(hidden.value)) {
            const parts = hidden.value.split("-");
            visibleInput.value = `${parts[2]}/${parts[1]}/${parts[0]}`;
        }
    }

    function syncDateInput(visibleInput, isBlur) {
        const hidden = findDateHiddenInput(visibleInput);
        if (!hidden) return;
        const text = visibleInput.value.trim();

        if (!text) {
            const wasVal = hidden.value;
            hidden.value = "";
            visibleInput.setCustomValidity("");
            if (wasVal) {
                hidden.dispatchEvent(new Event("change", { bubbles: true }));
            }
            validateDateRange(visibleInput.closest("form"));
            return;
        }

        const parsed = parseBrazilianDate(text);
        if (parsed.valid) {
            const wasVal = hidden.value;
            hidden.value = parsed.iso;
            visibleInput.setCustomValidity("");
            if (wasVal !== parsed.iso) {
                hidden.dispatchEvent(new Event("change", { bubbles: true }));
            }
            validateDateRange(visibleInput.closest("form"));
        } else if (parsed.incomplete) {
            hidden.value = "";
            if (isBlur) {
                visibleInput.setCustomValidity("Data incompleta. Informe a data completa no formato dd/mm/aaaa.");
            } else {
                visibleInput.setCustomValidity("");
            }
            validateDateRange(visibleInput.closest("form"));
        } else {
            hidden.value = "";
            visibleInput.setCustomValidity("Data inválida. Informe uma data de calendário válida.");
            validateDateRange(visibleInput.closest("form"));
        }
    }

    function validateDateRange(form) {
        if (!form) return;
        const startHidden = form.querySelector("[name=start_date]");
        const dueHidden = form.querySelector("[name=due_date]");
        if (!startHidden || !dueHidden) return;

        const dueVisible = form.querySelector("[data-date-target=" + (dueHidden.id || "") + "]") ||
            form.querySelector("#followup-due-date") ||
            (dueHidden.parentElement ? dueHidden.parentElement.querySelector("[data-date-input]") : null);

        if (!dueVisible) return;

        if (startHidden.value && dueHidden.value && /^\d{4}-\d{2}-\d{2}$/.test(startHidden.value) && /^\d{4}-\d{2}-\d{2}$/.test(dueHidden.value)) {
            if (dueHidden.value < startHidden.value) {
                dueVisible.setCustomValidity("A data limite não pode ser anterior à data de início");
            } else if (dueVisible.validationMessage === "A data limite não pode ser anterior à data de início") {
                dueVisible.setCustomValidity("");
            }
        } else if (dueVisible.validationMessage === "A data limite não pode ser anterior à data de início") {
            dueVisible.setCustomValidity("");
        }
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
        modal.querySelectorAll("[data-date-input]").forEach(function (dateInput) {
            initDateInput(dateInput);
            syncDateInput(dateInput, false);
        });

        const followUpForm = modal.querySelector("#followup-form");
        if (followUpForm) {
            const clientNameInput = followUpForm.querySelector("#client-name");
            const contactInput = followUpForm.querySelector("#client-contact");
            const isResolved = followUpForm.dataset.clientResolved === "true" || Boolean(clientNameInput && clientNameInput.dataset.selectedClientName);
            if (contactInput) {
                contactInput.disabled = !isResolved;
            }
            setFollowUpFieldsLocked(followUpForm, true);
        }

        const forms = modal.querySelectorAll("form");
        forms.forEach(validateDateRange);

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
        const duplicateDecision = form && form.querySelector("#duplicate-phone-decision");
        const duplicateToken = form && form.querySelector("#duplicate-phone-token");
        const confirmation = form && form.querySelector("#phone-change-confirmation");
        if (actionInput) {
            actionInput.value = "";
            delete actionInput.dataset.phoneChangeValue;
        }
        if (duplicateDecision) {
            duplicateDecision.value = "";
        }
        if (duplicateToken) {
            duplicateToken.value = "";
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
        if (!form || form.id !== "followup-form") {
            return;
        }
        if (idInput && idInput.value &&
                normalizeName(nameInput.value) === normalizeName(nameInput.dataset.selectedClientName || "")) {
            return;
        }

        if (idInput) {
            idInput.value = "";
        }
        if (contactInput) {
            if (contactInput.dataset.autofilledContact !== undefined &&
                    contactInput.value === contactInput.dataset.autofilledContact) {
                contactInput.value = "";
            }
            contactInput.disabled = true;
            delete contactInput.dataset.autofilledContact;
            delete contactInput.dataset.originalPhone;
        }
        delete nameInput.dataset.selectedClientName;
        clearPhoneDecision(form);
        setClientResolution(form, "");
        form.dataset.clientResolved = "false";
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
        contactInput.disabled = false;
        if (!preserveManualContact || !contactInput.value) {
            contactInput.value = client.contact;
            formatPhoneInput(contactInput);
            contactInput.dataset.autofilledContact = contactInput.value;
            contactInput.dataset.originalPhone = contactInput.value;
        } else {
            delete contactInput.dataset.autofilledContact;
        }
        clearClientMatches(form);
        clearPhoneDecision(form);
        setClientResolution(form, "existing");
        form.dataset.clientResolved = "true";
        setFollowUpFieldsLocked(form, true);
        contactInput.focus();
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
        delete contactInput.dataset.originalPhone;
        contactInput.disabled = false;
        clearClientMatches(form);
        clearPhoneDecision(form);
        setClientResolution(form, "new_homonym");
        form.dataset.clientResolved = "true";
        setFollowUpFieldsLocked(form, true);
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
        message.textContent = "Existe mais de um cliente com este nome. Escolha o telefone correto:";
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
        newClientButton.textContent = "Cadastrar outro cliente com este nome";
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
            form.dataset.clientResolved = "false";
            const contactInput = form.querySelector("#client-contact");
            if (contactInput) {
                contactInput.disabled = true;
            }
            setFollowUpFieldsLocked(form, true);
            return;
        }
        const idInput = form.querySelector("#client-id");
        const contactInput = form.querySelector("#client-contact");
        if (idInput && idInput.value &&
                normalizeName(nameInput.dataset.selectedClientName || "") === normalizeName(requestedName)) {
            setClientResolution(form, "existing");
            form.dataset.clientResolved = "true";
            if (contactInput) {
                contactInput.disabled = false;
                if (focusNext) {
                    contactInput.focus();
                }
            }
            setFollowUpFieldsLocked(form, true);
            return;
        }

        const contactAtRequest = contactInput ? contactInput.value : "";
        setClientResolution(form, "");
        form.dataset.clientResolved = "false";
        if (contactInput) {
            contactInput.disabled = true;
        }
        setFollowUpFieldsLocked(form, true);
        nameInput.setAttribute("aria-busy", "true");

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
                if (focusNext && contactInput) {
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
                form.dataset.clientResolved = "true";
                if (contactInput) {
                    contactInput.disabled = false;
                    delete contactInput.dataset.autofilledContact;
                    delete contactInput.dataset.originalPhone;
                    if (focusNext) {
                        contactInput.focus();
                    }
                }
                setFollowUpFieldsLocked(form, true);
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
                error.textContent = "Não foi possível verificar o cliente agora. Tente novamente antes de continuar.";
                message.replaceChildren(error);
            }
        });
    }

    function verifyClientPhone(form, focusNext) {
        if (!form || form.id !== "followup-form") {
            return;
        }
        const contactInput = form.querySelector("#client-contact");
        if (!contactInput || contactInput.disabled) {
            return;
        }
        const contactValue = contactInput.value.trim();
        if (contactValue.length < 15) {
            setFollowUpFieldsLocked(form, true);
            return;
        }

        const clientId = form.querySelector("#client-id") ? form.querySelector("#client-id").value : "";
        const clientName = form.querySelector("#client-name") ? form.querySelector("#client-name").value : "";
        const clientResolution = form.querySelector("#client-resolution") ? form.querySelector("#client-resolution").value : "";
        const phoneChangeAction = form.querySelector("#phone-change-action") ? form.querySelector("#phone-change-action").value : "";
        const duplicatePhoneDecision = form.querySelector("#duplicate-phone-decision") ? form.querySelector("#duplicate-phone-decision").value : "";
        const duplicatePhoneToken = form.querySelector("#duplicate-phone-token") ? form.querySelector("#duplicate-phone-token").value : "";
        const confirmationContainer = form.querySelector("#phone-change-confirmation");

        const requestSequence = ++phoneLookupSequence;
        const params = new URLSearchParams({
            client_id: clientId,
            client_name: clientName,
            contact: contactValue,
            client_resolution: clientResolution,
            phone_change_action: phoneChangeAction,
            duplicate_phone_decision: duplicatePhoneDecision,
            duplicate_phone_token: duplicatePhoneToken
        });

        fetch("/clients/phone-change-confirmation?" + params.toString(), {
            headers: { "HX-Request": "true" }
        }).then(function (response) {
            if (!response.ok) {
                throw new Error("phone verification failed");
            }
            return response.text();
        }).then(function (html) {
            if (requestSequence !== phoneLookupSequence || !form.isConnected) {
                return;
            }
            const trimmed = html.trim();
            if (trimmed) {
                if (confirmationContainer) {
                    confirmationContainer.innerHTML = trimmed;
                }
                setFollowUpFieldsLocked(form, true);
            } else {
                if (confirmationContainer) {
                    confirmationContainer.replaceChildren();
                }
                setFollowUpFieldsLocked(form, false);
                if (focusNext) {
                    const startDateInput = form.querySelector("#followup-start-date");
                    if (startDateInput && !startDateInput.disabled) {
                        startDateInput.focus();
                    } else {
                        const firstField = form.querySelector("#followup-fields input:not([type=hidden]), #followup-fields select");
                        if (firstField) {
                            firstField.focus();
                        }
                    }
                }
            }
        }).catch(function () {
            if (requestSequence !== phoneLookupSequence || !form.isConnected) {
                return;
            }
            setFollowUpFieldsLocked(form, true);
        });
    }

    function openConfirmModal(trigger) {
        const title = trigger.dataset.confirmTitle || "Confirmar ação";
        const message = trigger.dataset.confirmMessage || "Tem certeza que deseja continuar?";
        const actionText = trigger.dataset.confirmActionText || "Confirmar";
        const buttonClass = trigger.dataset.confirmButtonClass || "primary";
        const url = trigger.dataset.confirmUrl;
        if (!url) {
            return;
        }

        closeConfirmModal();

        const overlay = document.createElement("div");
        overlay.className = "modal-overlay confirm-modal-overlay";
        overlay.setAttribute("role", "presentation");
        overlay.style.zIndex = "1200";

        const section = document.createElement("section");
        section.className = "modal-content confirm-modal";
        section.setAttribute("role", "dialog");
        section.setAttribute("aria-modal", "true");
        section.setAttribute("aria-labelledby", "confirm-dialog-title");

        section.innerHTML = `
            <header class="modal-header">
                <div><h2 id="confirm-dialog-title">${title}</h2></div>
                <button type="button" class="icon-button confirm-modal-close" aria-label="Fechar">×</button>
            </header>
            <p>${message}</p>
            <div class="confirm-modal-actions">
                <button type="button" class="button ghost confirm-modal-close">Cancelar</button>
                <form hx-post="${url}" hx-target="#flash" hx-swap="innerHTML" style="margin:0">
                    <button type="submit" class="button ${buttonClass}">${actionText}</button>
                </form>
            </div>
        `;

        overlay.appendChild(section);

        const container = modalContainer();
        if (container) {
            container.appendChild(overlay);
            if (window.htmx) {
                window.htmx.process(section);
            }
            document.body.classList.add("modal-open");
            const submitBtn = section.querySelector("button[type=submit]");
            if (submitBtn) {
                submitBtn.focus();
            }
        }
    }

    function handlePrint() {
        originalDocumentTitle = document.title;
        const now = new Date();
        const year = now.getFullYear();
        const month = String(now.getMonth() + 1).padStart(2, "0");
        const day = String(now.getDate()).padStart(2, "0");
        const todayISO = `${year}-${month}-${day}`;

        let slug = "todos";
        const reportForm = document.getElementById("report-filters");
        if (reportForm) {
            const dateFrom = (reportForm.querySelector("[name=date_from]")?.value || "").trim();
            const dateTo = (reportForm.querySelector("[name=date_to]")?.value || "").trim();
            const client = (reportForm.querySelector("[name=client]")?.value || "").trim();
            const forwardTo = (reportForm.querySelector("[name=forward_to]")?.value || "").trim();
            const priority = (reportForm.querySelector("[name=priority]")?.value || "").trim();
            const statusVal = (reportForm.querySelector("[name=status]")?.value || "").trim();
            const overdue = Boolean(reportForm.querySelector("[name=overdue]")?.checked);

            const hasOtherFilters = Boolean(dateFrom || dateTo || client || forwardTo || priority);

            if (hasOtherFilters || (statusVal && overdue)) {
                slug = "filtrado";
            } else if (overdue) {
                slug = "atrasadas";
            } else if (statusVal === "PENDING") {
                slug = "pendentes";
            } else if (statusVal === "COMPLETED") {
                slug = "finalizadas";
            } else if (statusVal === "ARCHIVED") {
                slug = "arquivados";
            } else if (statusVal) {
                slug = "filtrado";
            } else {
                slug = "todos";
            }
        } else {
            const searchInput = document.getElementById("dashboard-client-search");
            const searchVal = (searchInput?.value || "").trim();
            if (searchVal) {
                slug = "filtrado";
            } else {
                const activeFilter = document.querySelector(".filter-bar .active, .filter-chip.active");
                if (activeFilter) {
                    const text = activeFilter.textContent.trim().toLowerCase();
                    if (text.includes("pendent")) slug = "pendentes";
                    else if (text.includes("finaliz")) slug = "finalizadas";
                    else if (text.includes("arquiv")) slug = "arquivados";
                    else if (text.includes("atras")) slug = "atrasadas";
                    else if (text.includes("tod")) slug = "todos";
                    else slug = "filtrado";
                }
            }
        }

        document.title = `client-follow-up_${slug}_${todayISO}`;
        window.print();
    }

    window.addEventListener("afterprint", function () {
        document.title = originalDocumentTitle;
    });

    document.addEventListener("keydown", function (event) {
        if (event.key === "Escape") {
            if (closeConfirmModal()) {
                return;
            }
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
        if (event.key === "Tab" && !event.shiftKey && event.target.id === "client-contact") {
            const form = event.target.closest("#followup-form");
            if (form) {
                form.dataset.tabAdvancing = "true";
            }
        }
    });

    document.addEventListener("click", function (event) {
        if (event.target.closest(".confirm-modal-close")) {
            closeConfirmModal();
            return;
        }

        if (event.target.closest(".modal-close")) {
            closeModal();
            return;
        }

        if (event.target.classList.contains("confirm-modal-overlay")) {
            closeConfirmModal();
            return;
        }

        if (event.target.classList.contains("modal-overlay")) {
            const hasActiveForm = event.target.querySelector("#followup-form, #followup-edit-form, #client-edit-region.client-edit");
            if (!hasActiveForm) {
                closeModal();
            }
            return;
        }

        const confirmTrigger = event.target.closest("[data-confirm-modal]");
        if (confirmTrigger) {
            openConfirmModal(confirmTrigger);
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
            return;
        }

        const newClient = event.target.closest(".client-match-new");
        if (newClient) {
            chooseNewHomonymousClient(newClient.closest("form"));
            return;
        }

        // Decisão de alteração de telefone (update ou new_client)
        const phoneDecision = event.target.closest("[data-phone-change-action]");
        if (phoneDecision) {
            const form = phoneDecision.closest("form");
            const actionInput = form && form.querySelector("#phone-change-action");
            const contactInput = form && form.querySelector("#client-contact");
            const duplicateDecision = form && form.querySelector("#duplicate-phone-decision");
            const duplicateToken = form && form.querySelector("#duplicate-phone-token");
            if (actionInput && contactInput) {
                actionInput.value = phoneDecision.dataset.phoneChangeAction;
                actionInput.dataset.phoneChangeValue = contactInput.value;
                if (duplicateDecision) duplicateDecision.value = "";
                if (duplicateToken) duplicateToken.value = "";
                phoneDecision.closest(".phone-confirmation").remove();
                setFollowUpFieldsLocked(form, true);
                verifyClientPhone(form, false);
            }
            return;
        }

        // Decisão de telefone duplicado ("Cadastrar mesmo assim")
        const duplicateAction = event.target.closest("[data-phone-duplicate-action]");
        if (duplicateAction) {
            const form = duplicateAction.closest("form");
            const decisionInput = form && (form.querySelector("#duplicate-phone-decision") || form.querySelector("#client-duplicate-phone-decision"));
            const tokenInput = form && (form.querySelector("#duplicate-phone-token") || form.querySelector("#client-duplicate-phone-token"));
            if (decisionInput) {
                decisionInput.value = duplicateAction.dataset.phoneDuplicateAction;
            }
            if (tokenInput && duplicateAction.dataset.duplicateToken) {
                tokenInput.value = duplicateAction.dataset.duplicateToken;
            }
            duplicateAction.closest(".phone-confirmation").remove();
            if (form && form.id === "followup-form") {
                setFollowUpFieldsLocked(form, true);
                verifyClientPhone(form, false);
            }
            return;
        }

        // Cancelar duplicidade
        const cancelDuplicate = event.target.closest("[data-cancel-phone-duplicate]");
        if (cancelDuplicate) {
            const form = cancelDuplicate.closest("form");
            const phoneInput = form && (form.querySelector("#client-contact") || form.querySelector("[name=contact]"));
            const origPhone = cancelDuplicate.dataset.originalPhone || (phoneInput && phoneInput.dataset.originalPhone) || "";
            const decisionInput = form && (form.querySelector("#duplicate-phone-decision") || form.querySelector("#client-duplicate-phone-decision"));
            const tokenInput = form && (form.querySelector("#duplicate-phone-token") || form.querySelector("#client-duplicate-phone-token"));
            const actionInput = form && form.querySelector("#phone-change-action");

            if (decisionInput) decisionInput.value = "";
            if (tokenInput) tokenInput.value = "";
            if (actionInput) {
                actionInput.value = "";
                delete actionInput.dataset.phoneChangeValue;
            }
            cancelDuplicate.closest(".phone-confirmation").remove();

            if (form && form.id === "followup-form") {
                if (origPhone) {
                    phoneInput.value = origPhone;
                    formatPhoneInput(phoneInput);
                    setFollowUpFieldsLocked(form, true);
                    verifyClientPhone(form, false);
                } else {
                    phoneInput.value = "";
                    setFollowUpFieldsLocked(form, true);
                    phoneInput.focus();
                }
            } else if (phoneInput) {
                phoneInput.value = origPhone;
                formatPhoneInput(phoneInput);
            }
            return;
        }

        // Cancelar alteração de telefone em editar cliente
        const cancelClientPhoneChange = event.target.closest("[data-cancel-client-phone-change]");
        if (cancelClientPhoneChange) {
            const form = cancelClientPhoneChange.closest("form");
            const phoneInput = form && form.querySelector("[name=contact]");
            const confirmationInput = form && form.querySelector("#client-phone-change-confirmation");
            if (phoneInput) {
                phoneInput.value = phoneInput.dataset.originalPhone || phoneInput.value;
                formatPhoneInput(phoneInput);
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
            handlePrint();
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
        if (event.target.matches("[data-date-input]")) {
            formatDateInput(event.target);
            syncDateInput(event.target, false);
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
            const duplicateDecision = form && form.querySelector("#client-duplicate-phone-decision");
            const duplicateToken = form && form.querySelector("#client-duplicate-phone-token");
            const message = form && form.querySelector("#client-edit-message");
            if (confirmationInput) confirmationInput.value = "";
            if (duplicateDecision) duplicateDecision.value = "";
            if (duplicateToken) duplicateToken.value = "";
            if (message) message.replaceChildren();
        }
        if (event.target.id === "client-contact") {
            const form = event.target.closest("form");
            clearPhoneDecision(form);
            setFollowUpFieldsLocked(form, true);
            if (event.target.dataset.autofilledContact !== undefined &&
                    event.target.value !== event.target.dataset.autofilledContact) {
                delete event.target.dataset.autofilledContact;
            }
        }
    }, true);

    document.addEventListener("change", function (event) {
        if (event.target.matches("[data-date-input]")) {
            syncDateInput(event.target, true);
        }
    }, true);

    document.addEventListener("focusout", function (event) {
        if (event.target.matches("[data-date-input]")) {
            syncDateInput(event.target, true);
        }
        if (event.target.id === "client-name") {
            const form = event.target.closest("#followup-form");
            if (form && form.dataset.clientResolved !== "true") {
                verifyExactClient(event.target, false);
            }
        }
        if (event.target.id === "client-contact") {
            const form = event.target.closest("#followup-form");
            const tabAdvancing = form && form.dataset.tabAdvancing === "true";
            if (form) {
                delete form.dataset.tabAdvancing;
            }
            if (form && event.target.value.length === 15) {
                verifyClientPhone(form, tabAdvancing);
            }
        }
    });

    document.addEventListener("htmx:afterRequest", function (event) {
        if (event.detail.successful && event.detail.elt.closest(".confirm-modal")) {
            closeConfirmModal();
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

    // Inicialização direta para páginas estáticas (ex.: relatórios)
    document.querySelectorAll("[data-date-input]").forEach(function (dateInput) {
        initDateInput(dateInput);
        syncDateInput(dateInput, false);
    });

    const container = modalContainer();
    if (container) {
        new MutationObserver(prepareModal).observe(container, { childList: true, subtree: true });
    }
}());
