package template

const UserForgotPassword = `<p>Hi {{replace.name}},</p>
<p>We accept requests to reset your account password. Please continue the password reset process by clicking the button below.</p>
<button style="
        border: none;
        padding: 10px;
        text-align: center;
        display: inline-block;
        font-size: 12px;
        margin: 4px 2px;
        cursor: pointer;
        background-color: #2F70ED;
        border-radius: 8px;
        ">
            <a  href="{{replace.url}}" style="color: white; text-decoration: none;">
                Reset password
            </a>
        </button>
<p>or you can click the link below:</p>
<p>{{replace.url}}</p>
<p>Your verification code:</p>
<h2>{{replace.otp}}</h2>
<br>
<p>If this is not you, please ignore this email.</p>
<p>Regards,<br>Gurih Mart.</p>`
