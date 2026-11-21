// The functions 30-date-groups passes in its props beside the data.
//
// formatDay turns an ISO date into the heading of a day: "2026-11-21" gives
// "November 21, Saturday". It finds the weekday in UTC, so the result does not
// depend on the time zone or the locale of the machine.
const months = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];
const weekdays = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

export default {
  formatDay(iso) {
    const [year, month, day] = iso.split("-").map(Number);
    const weekday = weekdays[new Date(Date.UTC(year, month - 1, day)).getUTCDay()];
    return `${months[month - 1]} ${day}, ${weekday}`;
  },
};
