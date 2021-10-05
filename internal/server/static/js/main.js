const cmdMap = {
  "show tables": `SELECT name FROM sqlite_master WHERE type = 'table';`,
};

const setupShell = (el) => {
  if (!el) {
    return;
  }
  el.focus();

  // move cursor to end
  let v = el.value; //store the value of the element
  el.value = ""; //clear the value of the element
  el.value = v; //set that value back.

  el.onkeyup = async (ke) => {
    if (ke.code === "Enter") {
      document.querySelectorAll(".shell-result").forEach((e) => {
        e.classList.remove("hide");
      });

      try {
        let cmd = el.value;
        cmd = cmdMap[cmd.toLowerCase().trim()] || cmd;

        const url = new URL(`${window.location.origin}/q`);
        url.searchParams.append("q", cmd);
        url.searchParams.append("db", window.DB_NAME);
        url.searchParams.append("fmt", "col");

        console.log(url.href);

        const start = new Date().getTime();
        const res = await fetch(url.href);
        const txt = await res.text();
        const end = new Date().getTime();

        document.getElementById("terminal-result").textContent = txt;
        document.getElementById("terminal-status").textContent = `status=[${res.status} ${res.statusText}] rtt=${end - start}ms`;
      } catch (e) {
        document.getElementById("terminal-result").textContent = `${e}`;
      }
    }
  };
};

setupShell(document.getElementById("mksql-shell"));
