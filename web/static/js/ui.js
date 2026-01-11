// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0
// EchoFeed UI helpers (Shadcn-style components)
// @author Anner
// Created on 2026/1/10

(function () {
  const TAG_COLOR_PALETTE = [
    { bg: "#F8FAFC", text: "#334155", border: "#E2E8F0" }, // slate
    { bg: "#FAFAFA", text: "#3F3F46", border: "#E4E4E7" }, // zinc
    { bg: "#FAFAF9", text: "#44403C", border: "#E7E5E4" }, // stone
    { bg: "#EFF6FF", text: "#1D4ED8", border: "#BFDBFE" }, // blue
    { bg: "#ECFEFF", text: "#0E7490", border: "#A5F3FC" }, // cyan
    { bg: "#ECFDF5", text: "#047857", border: "#A7F3D0" }, // emerald
    { bg: "#FFFBEB", text: "#B45309", border: "#FDE68A" }, // amber
    { bg: "#FFF1F2", text: "#BE123C", border: "#FECDD3" }, // rose
    { bg: "#F5F3FF", text: "#6D28D9", border: "#DDD6FE" }, // violet
  ];

  function hashString(str) {
    str = (str || "").toString();
    let h = 5381;
    for (let i = 0; i < str.length; i++) {
      h = (h * 33) ^ str.charCodeAt(i);
    }
    return h >>> 0;
  }

  function getTagColor(tag) {
    const idx = hashString(tag) % TAG_COLOR_PALETTE.length;
    return TAG_COLOR_PALETTE[idx];
  }

  function applyTagBadgeColors(root) {
    const scope = root || document;
    scope.querySelectorAll('[data-tag-badge="1"]').forEach((el) => {
      const tag = el.getAttribute("data-tag") || "";
      const c = getTagColor(tag);
      el.style.backgroundColor = c.bg;
      el.style.color = c.text;
      el.style.borderColor = c.border;

      const computed = window.getComputedStyle(el);
      if (computed.borderStyle === "none" || computed.borderWidth === "0px") {
        el.style.borderStyle = "solid";
        el.style.borderWidth = "1px";
      }
    });
  }

  function initShadcnTooltips(root) {
    const scope = root || document;

    function migrateTitles(node) {
      const nodes = [];
      if (node && node.nodeType === 1) nodes.push(node);
      if (node && node.querySelectorAll) {
        node.querySelectorAll("[title]").forEach((el) => nodes.push(el));
      }

      nodes.forEach((el) => {
        if (!el || !el.getAttribute) return;
        const title = el.getAttribute("title");
        if (!title) return;
        if (!el.dataset.tooltip) {
          el.dataset.tooltip = title;
        }
        el.removeAttribute("title");
        if (!el.getAttribute("aria-label")) {
          el.setAttribute("aria-label", el.dataset.tooltip || "");
        }
      });
    }

    migrateTitles(scope);

    let tooltipEl = null;
    let activeEl = null;
    let hideTimer = null;

    function ensureTooltipEl() {
      if (tooltipEl) return tooltipEl;
      const el = document.createElement("div");
      el.dataset.shadcnTooltip = "1";
      el.style.position = "fixed";
      el.style.left = "0";
      el.style.top = "0";
      el.style.zIndex = "9999";
      el.style.pointerEvents = "none";
      el.style.padding = "6px 10px";
      el.style.borderRadius = "8px";
      el.style.border = "1px solid hsl(240 5.9% 90%)";
      el.style.background = "hsl(0 0% 100%)";
      el.style.color = "hsl(240 10% 3.9%)";
      el.style.boxShadow = "0 10px 25px rgba(0,0,0,.10)";
      el.style.fontSize = "12px";
      el.style.lineHeight = "16px";
      el.style.maxWidth = "360px";
      el.style.whiteSpace = "pre-wrap";
      el.style.display = "none";
      el.style.opacity = "0";
      el.style.transition = "opacity .12s ease";
      document.body.appendChild(el);
      tooltipEl = el;
      return el;
    }

    function positionTooltipFor(el) {
      const tip = ensureTooltipEl();
      const rect = el.getBoundingClientRect();

      tip.style.display = "block";
      tip.style.visibility = "hidden";
      tip.style.left = "0px";
      tip.style.top = "0px";

      const tipRect = tip.getBoundingClientRect();
      const vw = window.innerWidth;
      const vh = window.innerHeight;
      const offset = 10;

      let left = rect.left + rect.width / 2 - tipRect.width / 2;
      left = Math.max(8, Math.min(vw - tipRect.width - 8, left));

      let top = rect.top - tipRect.height - offset;
      if (top < 8) {
        top = rect.bottom + offset;
      }
      if (top + tipRect.height > vh - 8) {
        top = Math.max(8, vh - tipRect.height - 8);
      }

      tip.style.left = Math.round(left) + "px";
      tip.style.top = Math.round(top) + "px";
      tip.style.visibility = "visible";
    }

    function showTooltip(el) {
      if (!el) return;
      const text = (el.dataset && el.dataset.tooltip) || "";
      if (!text) return;

      if (hideTimer) {
        clearTimeout(hideTimer);
        hideTimer = null;
      }

      activeEl = el;
      const tip = ensureTooltipEl();
      tip.textContent = text;
      positionTooltipFor(el);
      requestAnimationFrame(() => {
        if (!tooltipEl || activeEl !== el) return;
        tooltipEl.style.opacity = "1";
      });
    }

    function hideTooltip() {
      if (!tooltipEl) return;
      activeEl = null;
      tooltipEl.style.opacity = "0";
      hideTimer = setTimeout(() => {
        if (!tooltipEl) return;
        if (activeEl) return;
        tooltipEl.style.display = "none";
      }, 120);
    }

    function findTooltipTarget(fromEl) {
      if (!fromEl || !fromEl.closest) return null;
      return fromEl.closest("[data-tooltip]");
    }

    document.addEventListener(
      "mouseover",
      (e) => {
        const target = findTooltipTarget(e.target);
        if (!target) return;
        if (activeEl === target) return;
        showTooltip(target);
      },
      true,
    );

    document.addEventListener(
      "mouseout",
      (e) => {
        const target = findTooltipTarget(e.target);
        if (!target) return;
        const to = e.relatedTarget;
        if (to && target.contains(to)) return;
        if (activeEl === target) hideTooltip();
      },
      true,
    );

    document.addEventListener(
      "focusin",
      (e) => {
        const target = findTooltipTarget(e.target);
        if (!target) return;
        showTooltip(target);
      },
      true,
    );

    document.addEventListener(
      "focusout",
      (e) => {
        const target = findTooltipTarget(e.target);
        if (!target) return;
        if (activeEl === target) hideTooltip();
      },
      true,
    );

    window.addEventListener("scroll", hideTooltip, true);
    window.addEventListener("resize", hideTooltip, true);
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape") hideTooltip();
    });

    const mo = new MutationObserver((mutations) => {
      mutations.forEach((m) => {
        m.addedNodes.forEach((n) => migrateTitles(n));
        if (m.type === "attributes" && m.attributeName === "title") {
          migrateTitles(m.target);
        }
      });
    });
    try {
      mo.observe(document.documentElement, {
        subtree: true,
        childList: true,
        attributes: true,
        attributeFilter: ["title"],
      });
    } catch (_) {}
  }

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
    document.querySelectorAll('[data-shadcn-menu="1"]').forEach((m) => {
      if (m !== exceptMenu) m.classList.add("hidden");
    });
  }

  function closeAllShadcnMenus(exceptMenu) {
    closeAllShadcnSelectMenus(exceptMenu);
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
    menu.dataset.shadcnMenu = "1";

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
      closeAllShadcnMenus(menu);
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

  function createShadcnTagMultiSelect(opts) {
    const mount = opts && opts.mount;
    if (!mount) return null;

    mount.innerHTML = "";

    const placeholder = (opts && opts.placeholder) || "选择标签";
    const inputName = opts && opts.inputName;
    const inputContainer = opts && opts.inputContainer;
    const onChange = (opts && opts.onChange) || null;

    let options = Array.isArray(opts && opts.options) ? opts.options.slice() : [];
    let selected = Array.isArray(opts && opts.selected) ? opts.selected.slice() : [];

    function normalizeTag(s) {
      return (s || "")
        .toString()
        .replace(/\n/g, " ")
        .trim()
        .replace(/\s+/g, " ")
        .slice(0, 30);
    }

    function normalizeList(list) {
      const seen = new Set();
      const out = [];
      (list || []).forEach((t) => {
        t = normalizeTag(t);
        if (!t) return;
        if (seen.has(t)) return;
        seen.add(t);
        out.push(t);
      });
      out.sort();
      return out;
    }

    options = normalizeList(options);
    selected = normalizeList(selected);

    const wrapper = document.createElement("div");
    wrapper.className = "relative w-full";

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
      "absolute z-50 mt-1 w-full rounded-md border border-input bg-background p-2 shadow-lg hidden";
    menu.dataset.shadcnMenu = "1";

    const search = document.createElement("input");
    search.type = "text";
    search.className = "input h-9";
    search.placeholder = "搜索或输入新标签，回车添加";
    search.autocomplete = "off";

    const list = document.createElement("div");
    list.className = "mt-2 max-h-60 overflow-auto rounded-md border border-input";

    const badges = document.createElement("div");
    badges.className = "mt-2 flex flex-wrap gap-2";

    function syncHiddenInputs() {
      if (!inputName || !inputContainer) return;
      inputContainer.innerHTML = "";
      selected.forEach((t) => {
        const input = document.createElement("input");
        input.type = "hidden";
        input.name = inputName;
        input.value = t;
        inputContainer.appendChild(input);
      });
    }

    function setLabel() {
      if (selected.length === 0) {
        label.textContent = placeholder;
        return;
      }
      label.textContent = `已选 ${selected.length} 个标签`;
    }

    function renderBadges() {
      badges.innerHTML = "";
      if (selected.length === 0) return;

      selected.forEach((t) => {
        const b = document.createElement("button");
        b.type = "button";
        b.className =
          "inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium";
        b.dataset.tagBadge = "1";
        b.dataset.tag = t;
        const txt = document.createElement("span");
        txt.textContent = t;
        const x = document.createElement("span");
        x.className = "text-muted-foreground";
        x.textContent = "×";
        b.appendChild(txt);
        b.appendChild(x);
        b.addEventListener("click", () => {
          selected = selected.filter((v) => v !== t);
          sync();
          render();
          if (onChange) onChange(selected.slice());
        });
        badges.appendChild(b);
      });
    }

    function toggleSelected(tag) {
      tag = normalizeTag(tag);
      if (!tag) return;
      if (!options.includes(tag)) {
        options.push(tag);
        options = normalizeList(options);
      }
      if (selected.includes(tag)) {
        selected = selected.filter((v) => v !== tag);
      } else {
        selected = normalizeList(selected.concat([tag]));
      }
      sync();
      render();
      if (onChange) onChange(selected.slice());
    }

    function render() {
      const q = normalizeTag(search.value).toLowerCase();
      list.innerHTML = "";

      const items = options.filter((t) => !q || t.toLowerCase().includes(q));
      const existsExact = q && options.some((t) => t.toLowerCase() === q);

      if (q && !existsExact) {
        const addBtn = document.createElement("button");
        addBtn.type = "button";
        addBtn.className =
          "flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-left text-sm outline-none transition-colors hover:bg-secondary focus:bg-secondary";
        addBtn.innerHTML = `<span class="truncate">添加标签：${q}</span><span class="ml-2 text-primary">+</span>`;
        addBtn.addEventListener("click", () => {
          toggleSelected(q);
          search.value = "";
          render();
          search.focus();
        });
        list.appendChild(addBtn);
      }

      if (items.length === 0) {
        const empty = document.createElement("div");
        empty.className = "px-2 py-3 text-sm text-muted-foreground";
        empty.textContent = "暂无匹配的标签";
        list.appendChild(empty);
        return;
      }

      items.forEach((t) => {
        const item = document.createElement("button");
        item.type = "button";
        item.className =
          "flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-left text-sm outline-none transition-colors hover:bg-secondary focus:bg-secondary";
        const text = document.createElement("span");
        text.className = "truncate flex items-center gap-2";
        const dot = document.createElement("span");
        dot.className = "inline-block h-2 w-2 rounded-full border";
        dot.dataset.tagBadge = "1";
        dot.dataset.tag = t;
        text.appendChild(dot);
        const label = document.createElement("span");
        label.className = "truncate";
        label.textContent = t;
        text.appendChild(label);
        const check = document.createElement("span");
        check.className = "ml-2 text-primary";
        check.innerHTML = selected.includes(t)
          ? '<svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 13l4 4L19 7"/></svg>'
          : "";
        item.appendChild(text);
        item.appendChild(check);
        item.addEventListener("click", () => toggleSelected(t));
        list.appendChild(item);
      });
    }

    function sync() {
      setLabel();
      renderBadges();
      syncHiddenInputs();
      applyTagBadgeColors(wrapper);
    }

    button.addEventListener("click", (e) => {
      e.preventDefault();
      const isHidden = menu.classList.contains("hidden");
      closeAllShadcnMenus(menu);
      if (isHidden) {
        menu.classList.remove("hidden");
        search.focus();
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

    search.addEventListener("input", () => render());
    search.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        e.preventDefault();
        const q = normalizeTag(search.value);
        if (!q) return;
        toggleSelected(q);
        search.value = "";
        render();
      }
    });

    menu.appendChild(search);
    menu.appendChild(list);
    wrapper.appendChild(button);
    wrapper.appendChild(menu);

    mount.appendChild(wrapper);
    mount.appendChild(badges);

    sync();
    render();

    return {
      getSelected: () => selected.slice(),
      setSelected: (tags) => {
        selected = normalizeList(tags || []);
        options = normalizeList(options.concat(selected));
        sync();
        render();
      },
      setOptions: (tags) => {
        options = normalizeList(tags || []);
        options = normalizeList(options.concat(selected));
        render();
      },
    };
  }

  function createShadcnTagInput(opts) {
    const mount = opts && opts.mount;
    if (!mount) return null;

    mount.innerHTML = "";

    let options = Array.isArray(opts && opts.options) ? opts.options.slice() : [];
    let selected = Array.isArray(opts && opts.selected) ? opts.selected.slice() : [];
    const placeholder = (opts && opts.placeholder) || "输入标签，回车添加";
    const onChange = (opts && opts.onChange) || null;

    function normalizeTag(s) {
      return (s || "")
        .toString()
        .replace(/\n/g, " ")
        .trim()
        .replace(/\s+/g, " ")
        .slice(0, 30);
    }

    function normalizeList(list) {
      const seen = new Set();
      const out = [];
      (list || []).forEach((t) => {
        t = normalizeTag(t);
        if (!t) return;
        if (seen.has(t)) return;
        seen.add(t);
        out.push(t);
      });
      out.sort();
      return out;
    }

    options = normalizeList(options);
    selected = normalizeList(selected);

    const wrapper = document.createElement("div");
    wrapper.style.position = "relative";
    wrapper.style.width = "100%";

    const input = document.createElement("input");
    input.type = "text";
    input.className = "input";
    input.placeholder = placeholder;
    input.autocomplete = "off";

    const dropdown = document.createElement("div");
    dropdown.className = "card";
    dropdown.style.position = "absolute";
    dropdown.style.left = "0";
    dropdown.style.right = "0";
    dropdown.style.top = "calc(100% + 6px)";
    dropdown.style.zIndex = "60";
    dropdown.style.padding = "8px";
    dropdown.style.maxHeight = "240px";
    dropdown.style.overflow = "auto";
    dropdown.style.display = "none";
    dropdown.dataset.shadcnMenu = "1";

    const badges = document.createElement("div");
    badges.style.marginTop = "8px";
    badges.style.display = "flex";
    badges.style.flexWrap = "wrap";
    badges.style.gap = "6px";

    function renderBadges() {
      badges.innerHTML = "";
      selected.forEach((t) => {
        const b = document.createElement("button");
        b.type = "button";
        b.dataset.tagBadge = "1";
        b.dataset.tag = t;
        b.textContent = t + " ×";
        b.style.borderRadius = "9999px";
        b.style.borderWidth = "1px";
        b.style.borderStyle = "solid";
        b.style.padding = "2px 10px";
        b.style.fontSize = "12px";
        b.style.lineHeight = "18px";
        b.style.cursor = "pointer";
        b.addEventListener("click", () => {
          selected = selected.filter((v) => v !== t);
          sync();
          if (onChange) onChange(selected.slice());
        });
        badges.appendChild(b);
      });
      applyTagBadgeColors(badges);
    }

    function openDropdown() {
      closeAllShadcnMenus(dropdown);
      dropdown.style.display = "block";
    }

    function closeDropdown() {
      dropdown.style.display = "none";
    }

    function toggleSelected(tag) {
      tag = normalizeTag(tag);
      if (!tag) return;

      if (!options.includes(tag)) {
        options = normalizeList(options.concat([tag]));
      }

      if (selected.includes(tag)) {
        selected = selected.filter((v) => v !== tag);
      } else {
        selected = normalizeList(selected.concat([tag]));
      }

      sync();
      if (onChange) onChange(selected.slice());
    }

    function renderDropdown() {
      const q = normalizeTag(input.value).toLowerCase();
      dropdown.innerHTML = "";

      const matched = options.filter((t) => !q || t.toLowerCase().includes(q));
      const existsExact = q && options.some((t) => t.toLowerCase() === q);

      if (q && !existsExact) {
        const addBtn = document.createElement("button");
        addBtn.type = "button";
        addBtn.className = "btn btn-outline";
        addBtn.style.width = "100%";
        addBtn.style.justifyContent = "space-between";
        addBtn.textContent = "添加标签：" + q;
        addBtn.addEventListener("click", () => {
          toggleSelected(q);
          input.value = "";
          renderDropdown();
          input.focus();
        });
        dropdown.appendChild(addBtn);
      }

      if (matched.length === 0) {
        const empty = document.createElement("div");
        empty.className = "text-sm text-muted-foreground";
        empty.style.padding = "6px 4px";
        empty.textContent = "暂无匹配的标签";
        dropdown.appendChild(empty);
        return;
      }

      matched.forEach((t) => {
        const item = document.createElement("button");
        item.type = "button";
        item.className = "btn btn-ghost";
        item.style.width = "100%";
        item.style.justifyContent = "space-between";
        item.style.padding = "6px 10px";
        item.style.borderRadius = "8px";

        const left = document.createElement("span");
        left.style.display = "inline-flex";
        left.style.alignItems = "center";
        left.style.gap = "8px";

        const dot = document.createElement("span");
        dot.dataset.tagBadge = "1";
        dot.dataset.tag = t;
        dot.style.display = "inline-block";
        dot.style.width = "8px";
        dot.style.height = "8px";
        dot.style.borderRadius = "9999px";
        dot.style.borderWidth = "1px";
        dot.style.borderStyle = "solid";

        const label = document.createElement("span");
        label.textContent = t;
        label.style.overflow = "hidden";
        label.style.textOverflow = "ellipsis";
        label.style.whiteSpace = "nowrap";

        left.appendChild(dot);
        left.appendChild(label);

        const right = document.createElement("span");
        right.className = "text-sm text-muted-foreground";
        right.textContent = selected.includes(t) ? "已选" : "";

        item.appendChild(left);
        item.appendChild(right);
        item.addEventListener("click", () => toggleSelected(t));
        dropdown.appendChild(item);
      });

      applyTagBadgeColors(dropdown);
    }

    function sync() {
      renderBadges();
      renderDropdown();
    }

    input.addEventListener("focus", () => {
      openDropdown();
      renderDropdown();
    });
    input.addEventListener("input", () => {
      openDropdown();
      renderDropdown();
    });
    input.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        e.preventDefault();
        const v = normalizeTag(input.value);
        if (!v) return;
        toggleSelected(v);
        input.value = "";
        renderDropdown();
        openDropdown();
      } else if (e.key === "Escape") {
        closeDropdown();
      }
    });

    document.addEventListener("click", (e) => {
      if (!wrapper.contains(e.target)) {
        closeDropdown();
      }
    });

    wrapper.appendChild(input);
    wrapper.appendChild(dropdown);
    mount.appendChild(wrapper);
    mount.appendChild(badges);

    sync();

    return {
      getSelected: () => selected.slice(),
      setSelected: (tags) => {
        selected = normalizeList(tags || []);
        options = normalizeList(options.concat(selected));
        sync();
      },
      setOptions: (tags) => {
        options = normalizeList(tags || []);
        options = normalizeList(options.concat(selected));
        sync();
      },
      focus: () => input.focus(),
    };
  }

  document.addEventListener("DOMContentLoaded", () => {
    disableHistoryAutocomplete(document);
    enhanceAllShadcnSelects(document);
    applyTagBadgeColors(document);
    initShadcnTooltips(document);
  });

  window.enhanceShadcnSelect = enhanceShadcnSelect;
  window.enhanceAllShadcnSelects = enhanceAllShadcnSelects;
  window.disableHistoryAutocomplete = disableHistoryAutocomplete;
  window.createShadcnTagMultiSelect = createShadcnTagMultiSelect;
  window.createShadcnTagInput = createShadcnTagInput;
  window.applyTagBadgeColors = applyTagBadgeColors;
  window.initShadcnTooltips = initShadcnTooltips;
})();
