package template

const UserVerifyAccount = `<p>Click the button below and Enter the verification code.</p>
<p>This action takes you to a secure page where you can change your password.</p>
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
<p>or you can click link below:</p>
<p>{{replace.url}}</p>
<p>Your verification code is</p>
<h2>{{replace.otp}}</h2>
`
