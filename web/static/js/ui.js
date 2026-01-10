// EchoFeed UI helpers (Shadcn-style components)
// @author Anner
// Created on 2026/1/10

(function () {
  function disableHistoryAutocomplete(root) {
    const scope = root || document;

    scope.querySelectorAll("form").forEach((form) => {
      if (!form.hasAttribute("autocomplete")) {
        form.setAttribute("autocomplete", "off");
      }
    });

    scope.querySelectorAll("input").forEach((input) => {
      const type = (input.getAttribute("type") || "text").toLowerCase();
      if (type === "hidden") return;
      if (input.hasAttribute("autocomplete")) return;

      if (type === "password") {
        input.setAttribute("autocomplete", "new-password");
      } else {
        input.setAttribute("autocomplete", "off");
      }
    });

    scope.querySelectorAll("textarea").forEach((ta) => {
      if (!ta.hasAttribute("autocomplete")) {
        ta.setAttribute("autocomplete", "off");
      }
    });
  }

  function getSelectEl(selectOrId) {
    if (!selectOrId) return null;
    if (typeof selectOrId === "string") {
      return document.getElementById(selectOrId);
    }
    return selectOrId;
  }

  function closeAllShadcnSelectMenus(exceptMenu) {
    document.querySelectorAll('[data-shadcn-select-menu="1"]').forEach((m) => {
      if (m !== exceptMenu) m.classList.add("hidden");
    });
  }

  function enhanceShadcnSelect(selectOrId) {
    const select = getSelectEl(selectOrId);
    if (!select) return;

    if (select.__shadcnSelect && typeof select.__shadcnSelect.refresh === "function") {
      select.__shadcnSelect.refresh();
      return;
    }

    const wrapper = document.createElement("div");
    wrapper.className = "relative";

    Array.from(select.classList).forEach((cls) => {
      if (cls.startsWith("w-") || cls.startsWith("min-w-") || cls.startsWith("max-w-")) {
        wrapper.classList.add(cls);
        select.classList.remove(cls);
      }
    });
    if (
      !Array.from(wrapper.classList).some(
        (c) => c.startsWith("w-") || c.startsWith("min-w-") || c.startsWith("max-w-"),
      )
    ) {
      wrapper.classList.add("w-full");
    }
    select.parentNode.insertBefore(wrapper, select);
    wrapper.appendChild(select);

    select.classList.add("sr-only");

    const button = document.createElement("button");
    button.type = "button";
    button.className =
      "flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm ring-offset-background transition-colors hover:bg-secondary focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2";

    const label = document.createElement("span");
    label.className = "truncate";
    const icon = document.createElement("span");
    icon.className = "ml-2 text-muted-foreground";
    icon.innerHTML =
      '<svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 9l6 6 6-6"></path></svg>';
    button.appendChild(label);
    button.appendChild(icon);

    const menu = document.createElement("div");
    menu.className =
      "absolute z-50 mt-1 w-full max-h-72 overflow-auto rounded-md border border-input bg-background p-1 shadow-lg hidden";
    menu.dataset.shadcnSelectMenu = "1";

    const optionButtons = [];
    const optionChecks = new Map();

    function getSelectedLabel() {
      const selected = select.options[select.selectedIndex];
      if (selected && selected.value !== "") {
        return selected.textContent || "";
      }
      const placeholderOpt = Array.from(select.options).find((o) => o.value === "");
      return (placeholderOpt && placeholderOpt.textContent) || "请选择";
    }

    function sync() {
      label.textContent = getSelectedLabel();
      optionButtons.forEach((btn) => {
        if (btn.dataset.value === select.value) {
          btn.classList.add("bg-secondary");
          const chk = optionChecks.get(btn);
          if (chk) chk.classList.remove("invisible");
        } else {
          btn.classList.remove("bg-secondary");
          const chk = optionChecks.get(btn);
          if (chk) chk.classList.add("invisible");
        }
      });

      if (select.disabled) {
        button.setAttribute("disabled", "disabled");
        button.classList.add("opacity-50", "cursor-not-allowed");
      } else {
        button.removeAttribute("disabled");
        button.classList.remove("opacity-50", "cursor-not-allowed");
      }
    }

    function rebuildMenu() {
      optionButtons.length = 0;
      optionChecks.clear();
      menu.innerHTML = "";
      Array.from(select.options).forEach((opt) => {
        const item = document.createElement("button");
        item.type = "button";
        item.className =
          "flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-left text-sm outline-none transition-colors hover:bg-secondary focus:bg-secondary";
        const text = document.createElement("span");
        text.className = "truncate";
        text.textContent = opt.textContent || "";
        const check = document.createElement("span");
        check.className = "ml-2 text-primary invisible";
        check.innerHTML =
          '<svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 13l4 4L19 7"/></svg>';
        item.appendChild(text);
        item.appendChild(check);

        item.dataset.value = opt.value;
        item.addEventListener("click", () => {
          select.value = opt.value;
          select.dispatchEvent(new Event("change", { bubbles: true }));
          menu.classList.add("hidden");
          sync();
        });
        menu.appendChild(item);
        optionButtons.push(item);
        optionChecks.set(item, check);
      });
      sync();
    }

    button.addEventListener("click", (e) => {
      e.preventDefault();
      if (select.disabled) return;
      const isHidden = menu.classList.contains("hidden");
      closeAllShadcnSelectMenus(menu);
      if (isHidden) {
        menu.classList.remove("hidden");
      } else {
        menu.classList.add("hidden");
      }
    });

    document.addEventListener("click", (e) => {
      if (!wrapper.contains(e.target)) {
        menu.classList.add("hidden");
      }
    });
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape") {
        menu.classList.add("hidden");
      }
    });

    select.addEventListener("change", () => sync());

    wrapper.appendChild(button);
    wrapper.appendChild(menu);

    select.__shadcnSelect = {
      refresh: rebuildMenu,
      sync,
      menu,
      button,
    };

    rebuildMenu();
  }

  function enhanceAllShadcnSelects(root) {
    const scope = root || document;
    scope.querySelectorAll("select[data-shadcn-select=\"1\"]").forEach((sel) => enhanceShadcnSelect(sel));
  }

  document.addEventListener("DOMContentLoaded", () => {
    disableHistoryAutocomplete(document);
    enhanceAllShadcnSelects(document);
  });

  window.enhanceShadcnSelect = enhanceShadcnSelect;
  window.enhanceAllShadcnSelects = enhanceAllShadcnSelects;
  window.disableHistoryAutocomplete = disableHistoryAutocomplete;
})();
