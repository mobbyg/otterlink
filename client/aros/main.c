/*
 * Otter Link AROS client — initial native GUI shell.
 *
 * This is a UI/build scaffold only. Networking and authentication are not
 * implemented yet. Keep the client connected directly to the existing
 * Otter Link server; do not add a local server or proxy.
 */

#include <exec/types.h>
#include <libraries/mui.h>

#include <proto/exec.h>
#include <proto/intuition.h>
#include <proto/muimaster.h>
#include <clib/alib_protos.h>

int main(void)
{
    Object *app;
    Object *window;
    Object *quit_button;
    ULONG signals = 0;

    app = ApplicationObject,
        MUIA_Application_Title, "Otter Link",
        MUIA_Application_Version, "$VER: OtterLink AROS 0.1 (12.10.2026)",
        SubWindow, window = WindowObject,
            MUIA_Window_Title, "Otter Link",
            MUIA_Window_ID, MAKE_ID('O', 'T', 'T', 'R'),
            WindowContents, VGroup,
                Child, TextObject,
                    MUIA_Text_Contents,
                        "Welcome to Otter Link for AROS\n\n"
                        "Native client foundation\n"
                        "Connection and sign-in are not implemented yet.",
                    MUIA_Text_PreParse, "\33c",
                    End,
                Child, HGroup,
                    Child, HVSpace,
                    Child, quit_button = SimpleButton("_Quit"),
                    Child, HVSpace,
                    End,
                End,
            End,
        End;

    if (app == NULL)
        return RETURN_FAIL;

    DoMethod(window, MUIM_Notify,
        MUIA_Window_CloseRequest, TRUE,
        (IPTR) app, 2,
        MUIM_Application_ReturnID, MUIV_Application_ReturnID_Quit);

    DoMethod(quit_button, MUIM_Notify,
        MUIA_Pressed, FALSE,
        (IPTR) app, 2,
        MUIM_Application_ReturnID, MUIV_Application_ReturnID_Quit);

    set(window, MUIA_Window_Open, TRUE);

    if (XGET(window, MUIA_Window_Open))
    {
        while (DoMethod(app, MUIM_Application_NewInput, (IPTR) &signals)
               != MUIV_Application_ReturnID_Quit)
        {
            if (signals != 0)
            {
                signals = Wait(signals | SIGBREAKF_CTRL_C);
                if (signals & SIGBREAKF_CTRL_C)
                    break;
            }
        }
    }

    MUI_DisposeObject(app);
    return RETURN_OK;
}
