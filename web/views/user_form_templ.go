package views

import templ "github.com/a-h/templ"

func UserForm(user *User) templ.Component {
	return templ.ComponentFunc(func(ctx templ.Context) (err error) {
		ctx.WriteString("<html><body>")
		ctx.WriteString("<h1>Edit User</h1>")
		ctx.WriteString("<form method='POST' action='/updateuser'>")
		ctx.WriteString("<label for='fullName'>Full Name:</label>")
		ctx.WriteString("<input type='text' id='fullName' name='fullName' value='" + user.FullName + "' readonly><br>")
		ctx.WriteString("<label for='trade'>Trade:</label>")
		ctx.WriteString("<input type='text' id='trade' name='trade' value='" + user.Trade + "'><br>")
		ctx.WriteString("<input type='submit' value='Update'>")
		ctx.WriteString("</form>")
		ctx.WriteString("</body></html>")
		return
	})
}
