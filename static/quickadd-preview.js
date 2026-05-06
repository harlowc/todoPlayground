(function () {
  var datePattern = /^\d{4}-\d{2}-\d{2}$/;
  var categoryPattern = /^@([A-Za-z0-9][A-Za-z0-9_-]*)$/;
  var priorityPattern = /^p([123])$/i;

  function cleanToken(token) {
    return token.replace(/^[\s.,;:!?()[\]{}"']+|[\s.,;:!?()[\]{}"']+$/g, "");
  }

  function cleanText(text) {
    return text.trim().replace(/^[,;:\s]+|[,;:\s]+$/g, "").replace(/\s+/g, " ");
  }

  function tokenize(text) {
    return text.split(/\s+/).map(function (word) {
      return { raw: word, clean: cleanToken(word) };
    }).filter(function (token) {
      return token.clean !== "";
    });
  }

  function addDays(isoDate, days) {
    var parts = isoDate.split("-").map(Number);
    var date = new Date(Date.UTC(parts[0], parts[1] - 1, parts[2]));
    date.setUTCDate(date.getUTCDate() + days);
    return date.toISOString().slice(0, 10);
  }

  function dateCue(word) {
    return ["by", "due", "on"].indexOf(word.toLowerCase()) !== -1;
  }

  function dateValueAt(tokens, index, today) {
    var token = tokens[index];
    if (!token) {
      return null;
    }
    if (token.clean.toLowerCase() === "today") {
      return { value: today, consumed: 1 };
    }
    if (token.clean.toLowerCase() === "tomorrow") {
      return { value: addDays(today, 1), consumed: 1 };
    }
    if (datePattern.test(token.clean)) {
      return { value: token.clean, consumed: 1 };
    }
    if (token.clean.toLowerCase() === "next" && tokens[index + 1] && tokens[index + 1].clean.toLowerCase() === "week") {
      return { value: addDays(today, 7), consumed: 2 };
    }
    return null;
  }

  function dateAt(tokens, index, today) {
    var token = tokens[index];
    if (dateCue(token.clean)) {
      var cued = dateValueAt(tokens, index + 1, today);
      if (!cued) {
        return null;
      }
      return { value: cued.value, consumed: cued.consumed + 1 };
    }
    if (index === 0) {
      return null;
    }
    return dateValueAt(tokens, index, today);
  }

  function categoryCue(word) {
    var clean = word.toLowerCase();
    return clean === "category" || clean === "label";
  }

  function priorityWord(word) {
    switch (word.toLowerCase()) {
    case "high":
      return "high";
    case "normal":
    case "medium":
      return "normal";
    case "low":
      return "low";
    default:
      return "";
    }
  }

  function priorityFromLevel(level) {
    switch (level) {
    case "1":
      return "high";
    case "2":
      return "normal";
    case "3":
      return "low";
    default:
      return "normal";
    }
  }

  function priorityAt(tokens, index) {
    var token = tokens[index];
    var match = token.clean.match(priorityPattern);
    if (match) {
      return { value: priorityFromLevel(match[1]), consumed: 1 };
    }

    var wordPriority = priorityWord(token.clean);
    if (wordPriority && tokens[index + 1] && tokens[index + 1].clean.toLowerCase() === "priority") {
      return { value: wordPriority, consumed: 2 };
    }

    if (token.clean.toLowerCase() === "priority" && tokens[index + 1]) {
      wordPriority = priorityWord(tokens[index + 1].clean);
      if (wordPriority) {
        return { value: wordPriority, consumed: 2 };
      }
    }
    return null;
  }

  function categoryBoundary(tokens, index, today) {
    var token = tokens[index];
    return categoryCue(token.clean) ||
      !!dateAt(tokens, index, today) ||
      !!priorityAt(tokens, index) ||
      categoryPattern.test(token.clean);
  }

  function categoryAt(tokens, index, today) {
    if (!categoryCue(tokens[index].clean) || !tokens[index + 1]) {
      return null;
    }

    var category = [];
    for (var i = index + 1; i < tokens.length; i++) {
      if (categoryBoundary(tokens, i, today)) {
        break;
      }
      category.push(tokens[i].clean);
      if (/[;,]$/.test(tokens[i].raw)) {
        i++;
        return { value: cleanText(category.join(" ")), consumed: i - index };
      }
    }

    if (!category.length) {
      return null;
    }
    return { value: cleanText(category.join(" ")), consumed: category.length + 1 };
  }

  function parseQuickAdd(text, today, explicit) {
    var tokens = tokenize(text);
    var kept = [];
    var parsed = {
      text: "",
      dueDate: explicit.dueDate || "",
      category: explicit.category || "",
      priority: explicit.priority || "normal",
      inferredDueDate: "",
      inferredCategory: "",
      inferredPriority: ""
    };

    for (var i = 0; i < tokens.length; i++) {
      var token = tokens[i];
      var date = dateAt(tokens, i, today);
      if (date) {
        if (!explicit.dueDate) {
          parsed.dueDate = date.value;
          parsed.inferredDueDate = date.value;
        }
        i += date.consumed - 1;
        continue;
      }

      var category = categoryAt(tokens, i, today);
      if (category) {
        if (!explicit.category) {
          parsed.category = category.value;
          parsed.inferredCategory = category.value;
        }
        i += category.consumed - 1;
        continue;
      }

      var categoryMatch = token.clean.match(categoryPattern);
      if (categoryMatch) {
        if (!explicit.category) {
          parsed.category = categoryMatch[1];
          parsed.inferredCategory = categoryMatch[1];
        }
        continue;
      }

      var priority = priorityAt(tokens, i);
      if (priority) {
        if (!explicit.priority || explicit.priority === "normal") {
          parsed.priority = priority.value;
          parsed.inferredPriority = priority.value;
        }
        i += priority.consumed - 1;
        continue;
      }

      kept.push(token.raw);
    }

    parsed.text = cleanText(kept.join(" "));
    return parsed;
  }

  function priorityLabel(priority) {
    switch (priority) {
    case "high":
      return "High priority";
    case "low":
      return "Low priority";
    default:
      return "Normal priority";
    }
  }

  function setChip(chip, value) {
    if (!chip) {
      return;
    }
    chip.textContent = value;
    chip.hidden = value === "";
  }

  function updatePreview(preview) {
    var form = preview.closest("form");
    if (!form) {
      return;
    }

    var textInput = form.querySelector('input[name="text"]');
    var dueInput = form.querySelector('input[name="due_date"]');
    var categoryInput = form.querySelector('input[name="category"]');
    var priorityInput = form.querySelector('select[name="priority"]');
    var today = preview.dataset.today;

    if (!textInput || !today) {
      return;
    }

    var parsed = parseQuickAdd(textInput.value, today, {
      dueDate: dueInput ? dueInput.value : "",
      category: categoryInput ? categoryInput.value.trim() : "",
      priority: priorityInput ? priorityInput.value : ""
    });

    var hasInference = parsed.inferredDueDate || parsed.inferredCategory || parsed.inferredPriority || parsed.text !== cleanText(textInput.value);
    preview.hidden = !hasInference;
    setChip(preview.querySelector("[data-preview-task]"), hasInference && parsed.text ? "Task: " + parsed.text : "");
    setChip(preview.querySelector("[data-preview-due]"), parsed.inferredDueDate ? "Due: " + parsed.inferredDueDate : "");
    setChip(preview.querySelector("[data-preview-category]"), parsed.inferredCategory ? "Category: " + parsed.inferredCategory : "");
    setChip(preview.querySelector("[data-preview-priority]"), parsed.inferredPriority ? priorityLabel(parsed.inferredPriority) : "");
  }

  function bindPreview(preview) {
    if (preview.dataset.quickAddPreviewBound === "true") {
      return;
    }
    preview.dataset.quickAddPreviewBound = "true";

    var form = preview.closest("form");
    if (!form) {
      return;
    }
    form.addEventListener("input", function () {
      updatePreview(preview);
    });
    form.addEventListener("change", function () {
      updatePreview(preview);
    });
    updatePreview(preview);
  }

  function bindAllPreviews() {
    document.querySelectorAll("[data-quick-add-preview]").forEach(bindPreview);
  }

  document.addEventListener("DOMContentLoaded", bindAllPreviews);
  document.addEventListener("htmx:afterSwap", bindAllPreviews);
})();
